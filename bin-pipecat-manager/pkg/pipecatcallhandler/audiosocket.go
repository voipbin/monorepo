package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_audiosocket.go -source audiosocket.go -build_flags=-mod=mod

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/zaf/resample"
)

type AudiosocketHandler interface {
	GetStreamingID(conn net.Conn) (uuid.UUID, error)
	GetNextMedia(conn net.Conn) (audiosocket.Message, error)
	GetDataSamples(inputRate int, data []byte) ([]byte, error)
	Upsample8kTo16k(data []byte) ([]byte, error)
	WrapDataPCM16Bit(data []byte) ([]byte, error)
	Write(ctx context.Context, conn net.Conn, data []byte) error
}

const (
	defaultAudiosocketFormatSLIN        = 0x10 // SLIN format for 16-bit PCM audio
	defaultAudiosocketMaxFragmentSize   = 320  // Maximum fragment size for Audiosocket messages
	defaultAudiosocketConvertSampleRate = 8000 // Default sample rate for conversion to 8kHz. This must not be changed as it is the minimum sample rate for audiosocket.
)

type audiosocketHandler struct{}

func NewAudiosocketHandler() AudiosocketHandler {
	return &audiosocketHandler{}
}

// GetStreamingID gets the streaming id from the connection
// the first message of the audiosocket should be the streaming id
func (h *audiosocketHandler) GetStreamingID(conn net.Conn) (uuid.UUID, error) {
	m, err := audiosocket.NextMessage(conn)
	if err != nil {
		return uuid.Nil, err
	}

	if m.Kind() != audiosocket.KindID {
		return uuid.Nil, fmt.Errorf("wrong message kind. kind: %v", m.Kind())
	}

	res := uuid.FromBytesOrNil(m.Payload())
	return res, nil
}

func (h *audiosocketHandler) GetNextMedia(conn net.Conn) (audiosocket.Message, error) {
	m, err := audiosocket.NextMessage(conn)
	if err != nil {
		return nil, err
	}

	if m.Kind() != audiosocket.KindSlin {
		if m.Kind() == audiosocket.KindHangup {
			return nil, fmt.Errorf("received hangup. content_len: %d, kind: %v", m.ContentLength(), m.Kind())
		}

		return nil, nil
	}

	if m.ContentLength() < 1 {
		return nil, nil
	}

	return m, nil
}

// GetDataSamples processes 16-bit PCM data with the given inputRate sample rate.
// It uses libsoxr (via zaf/resample) for high-quality resampling with proper anti-aliasing.
// If inputRate equals defaultConvertSampleRate (8kHz), it returns data as is.
func (h *audiosocketHandler) GetDataSamples(inputRate int, data []byte) ([]byte, error) {
	if inputRate == defaultAudiosocketConvertSampleRate {
		// No conversion needed
		return data, nil
	}

	if len(data) == 0 {
		return data, nil
	}

	// Get buffer from pool and estimate output size
	inputSamples := len(data) / 2
	outputSamples := inputSamples * defaultAudiosocketConvertSampleRate / inputRate
	output := getBuffer()
	output.Grow(outputSamples * 2)

	// Create resampler: input rate -> 8kHz, mono channel, I16 format, MediumQ quality
	resampler, err := resample.New(
		output,
		float64(inputRate),
		float64(defaultAudiosocketConvertSampleRate),
		1,                // mono
		resample.I16,     // 16-bit signed linear PCM
		resample.MediumQ, // balance quality vs CPU
	)
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to create resampler: %w", err)
	}

	// Write input data to the resampler
	_, err = resampler.Write(data)
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to write to resampler: %w", err)
	}

	// Close to flush any remaining output
	err = resampler.Close()
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to close resampler: %w", err)
	}

	// Copy result before returning buffer to pool
	result := make([]byte, output.Len())
	copy(result, output.Bytes())
	putBuffer(output)

	return result, nil
}

// Upsample8kTo16k performs a simple 2× upsampling from 8 kHz to 16 kHz.
//
// It assumes the input is 16-bit little-endian PCM mono audio (int16 per sample).
// The algorithm uses linear interpolation: for each original sample pair (s1, s2),
// it inserts one midpoint sample (average of s1 and s2), effectively doubling
// the sample rate. This produces smoother playback than simple duplication while
// remaining computationally lightweight.
//
// Note: This method is designed for low-latency real-time audio streaming, not
// high-fidelity resampling. For higher quality, consider using a windowed
func (h *audiosocketHandler) Upsample8kTo16k(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("the PCM data must be 16-bit aligned (even number of bytes). bytes: %d", len(data))
	}

	numSamples := len(data) / 2
	if numSamples == 0 {
		return []byte{}, nil
	}

	// Pre-allocate output buffer: (n-1)*2 + 1 samples = 2n-1 samples
	// Each sample is 2 bytes, so output size is (2*numSamples - 1) * 2
	outputSize := (2*numSamples - 1) * 2
	out := getBuffer()
	out.Grow(outputSize)

	for i := 0; i < numSamples-1; i++ {
		s1 := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
		s2 := int16(binary.LittleEndian.Uint16(data[(i+1)*2 : (i+1)*2+2]))

		_ = binary.Write(out, binary.LittleEndian, s1)
		mid := int16((int32(s1) + int32(s2)) / 2)
		_ = binary.Write(out, binary.LittleEndian, mid)
	}

	// Write last sample
	last := int16(binary.LittleEndian.Uint16(data[(numSamples-1)*2:]))
	_ = binary.Write(out, binary.LittleEndian, last)

	// Copy result before returning buffer to pool
	result := make([]byte, out.Len())
	copy(result, out.Bytes())
	putBuffer(out)

	return result, nil
}

// WrapDataPCM16Bit wraps raw 16-bit PCM audio data into the Audiosocket transmission format.
//
// The wrapped byte slice has the following structure:
//   - 2 bytes: Audio format identifier (uint16, BigEndian), fixed to audiosocketFormatSLIN (0x10 for signed linear PCM)
//   - 2 bytes: Sample count (uint16, BigEndian), representing the number of 16-bit samples
//   - N bytes: Raw audio payload (16-bit PCM data)
//
// Parameters:
//
//	data []byte - Raw audio data buffer, must be 16-bit aligned (even length) since each sample is 2 bytes.
//
// Returns:
//
//	[]byte - The wrapped byte slice ready to be sent via Audiosocket.
//	error  - Returns an error if the input data length is not valid or if writing to the buffer fails.
//
// Notes:
//   - The function expects input PCM data to be signed 16-bit samples (little or big endian doesn't matter here,
//     since this function only wraps data without conversion).
//   - The sample count is calculated as len(data) / 2 because each sample consists of 2 bytes.
//   - The resulting byte slice can be directly transmitted over Audiosocket protocol.
func (h *audiosocketHandler) WrapDataPCM16Bit(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("the PCM data must be 16-bit aligned (even number of bytes). bytes: %d", len(data))
	}

	// Header: 1 byte format + 2 bytes length + data
	headerSize := 3
	buf := getBuffer()
	buf.Grow(headerSize + len(data))

	// Write audio format (SLIN)
	if errWrite := buf.WriteByte(defaultAudiosocketFormatSLIN); errWrite != nil {
		putBuffer(buf)
		return nil, fmt.Errorf("failed to write data type: %w", errWrite)
	}

	// Write payload length
	payloadLength := uint16(len(data))
	if errWrite := binary.Write(buf, binary.BigEndian, payloadLength); errWrite != nil {
		putBuffer(buf)
		return nil, errors.Wrapf(errWrite, "could not write sample count")
	}

	// Write raw PCM data
	_, err := buf.Write(data)
	if err != nil {
		putBuffer(buf)
		return nil, errors.Wrapf(err, "could not write raw audio data")
	}

	// Copy result before returning buffer to pool
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	putBuffer(buf)

	return result, nil
}

// Write fragments and sends large 16-bit PCM audio data over an Audiosocket connection.
//
// Purpose:
//   - To avoid overwhelming the connection, this function splits the input audio data into smaller fragments,
//     wraps each fragment in the Audiosocket format, and writes them sequentially to the connection with a short delay between each write.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control.
//   - conn: The net.Conn connection to which audio data will be sent.
//   - data: Raw audio data as a byte slice (must be 16-bit PCM, i.e., even number of bytes).
//
// Behavior:
//   - The function divides the input data into fragments of up to audiosocketMaxFragmentSize bytes.
//   - Each fragment is wrapped using audiosocketWrapDataPCM16Bit before being sent.
//   - After each fragment is written, the function waits for audiosocketWriteDelay to prevent flooding the connection.
//   - If the context is cancelled or an error occurs during wrapping or writing, the function returns an error.
//
// Returns:
//   - error: Returns an error if the context is cancelled, the data is invalid, or writing fails.
func (h *audiosocketHandler) Write(ctx context.Context, conn net.Conn, data []byte) error {
	if len(data) == 0 {
		// nothing to send
		return nil
	}

	payloadLen := len(data)
	offset := 0

	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(defaultAudiosocketMaxFragmentSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		tmp, err := h.WrapDataPCM16Bit(fragment)
		if err != nil {
			return errors.Wrapf(err, "failed to wrap data for audiosocket")
		}

		_, err = conn.Write(tmp)
		if err != nil {
			return errors.Wrapf(err, "failed to write wrapped data to connection")
		}

		offset += fragmentLen
	}

	return nil
}
