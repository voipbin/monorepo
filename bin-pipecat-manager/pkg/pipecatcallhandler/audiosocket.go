package pipecatcallhandler

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultAudiosocketFormatSLIN        = 0x10                  // SLIN format for 16-bit PCM audio
	defaultAudiosocketMaxFragmentSize   = 320                   // Maximum fragment size for Audiosocket messages
	defaultAudiosocketWriteDelay        = 20 * time.Millisecond // Delay between writing fragments to avoid flooding the connection
	defaultAudiosocketConvertSampleRate = 8000                  // Default sample rate for conversion to 8kHz. This must not be changed as it is the minimum sample rate for audiosocket.
)

// audiosocketGetStreamingID gets the streaming id from the connection
// the first message of the audiosocket should be the streaming id
func audiosocketGetStreamingID(conn net.Conn) (uuid.UUID, error) {
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

func audiosocketGetNextMedia(conn net.Conn) (audiosocket.Message, error) {
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

// audiosocketGetDataSamples processes 16-bit PCM data with the given inputRate sample rate.
// If inputRate equals defaultConvertSampleRate, it returns data as is.
// If inputRate is an integer multiple of defaultConvertSampleRate, it downsamples accordingly.
// Otherwise, it returns an error because only integer downsampling is supported.
func audiosocketGetDataSamples(inputRate int, data []byte) ([]byte, error) {
	if inputRate == defaultAudiosocketConvertSampleRate {
		// No conversion needed
		return data, nil
	}

	if inputRate%defaultAudiosocketConvertSampleRate != 0 {
		return nil, fmt.Errorf("cannot convert %d Hz to %d Hz: only integer downsampling supported", inputRate, defaultAudiosocketConvertSampleRate)
	}

	factor := inputRate / defaultAudiosocketConvertSampleRate
	res := make([]byte, 0, len(data)/factor)

	// Downsample by selecting every 'factor'-th sample (2 bytes per sample)
	for i := 0; i+2*factor-1 < len(data); i += 2 * factor {
		res = append(res, data[i], data[i+1])
	}

	return res, nil
}

// audiosocketWrapDataPCM16Bit wraps raw 16-bit PCM audio data into the Audiosocket transmission format.
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
func audiosocketWrapDataPCM16Bit(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("the PCM data must be 16-bit aligned (even number of bytes). bytes: %d", len(data))
	}

	buf := new(bytes.Buffer)

	// Write audio format (SLIN)
	if errWrite := buf.WriteByte(defaultAudiosocketFormatSLIN); errWrite != nil {
		return nil, fmt.Errorf("failed to write data type: %w", errWrite)
	}

	// Write sample count
	payloadLength := uint16(len(data))
	if errWrite := binary.Write(buf, binary.BigEndian, payloadLength); errWrite != nil {
		return nil, errors.Wrapf(errWrite, "could not write sample count")
	}

	// Write raw PCM data
	_, err := buf.Write(data)
	if err != nil {
		return nil, errors.Wrapf(err, "could not write raw audio data")
	}

	return buf.Bytes(), nil
}

// audiosocketWrite fragments and sends large 16-bit PCM audio data over an Audiosocket connection.
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
func audiosocketWrite(ctx context.Context, conn net.Conn, data []byte) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "audiosocketWrite",
	})
	if len(data) == 0 {
		// nothing to send
		return nil
	}

	payloadLen := len(data)
	offset := 0

	log.Debugf("Sending %d bytes of audio data in fragments", payloadLen)
	for offset < payloadLen {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		fragmentLen := min(defaultAudiosocketMaxFragmentSize, payloadLen-offset)
		fragment := data[offset : offset+fragmentLen]

		tmp, err := audiosocketWrapDataPCM16Bit(fragment)
		if err != nil {
			return errors.Wrapf(err, "failed to wrap data for audiosocket")
		}

		_, err = conn.Write(tmp)
		if err != nil {
			return errors.Wrapf(err, "failed to write wrapped data to connection")
		}

		offset += fragmentLen

		select {
		case <-time.After(defaultAudiosocketWriteDelay):
			// do nothing
			continue

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
