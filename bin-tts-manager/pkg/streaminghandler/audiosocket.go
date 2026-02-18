package streaminghandler

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
	audiosocketFormatSLIN      = 0x10                  // SLIN format for 16-bit PCM audio
	audiosocketMaxFragmentSize  = 320                   // Maximum fragment size for Audiosocket messages
	audiosocketWriteDelay      = 20 * time.Millisecond // Delay between writing fragments to avoid flooding the connection
	audiosocketSilenceFrameSize = 320                   // 160 samples * 2 bytes/sample = 20ms at 8kHz 16-bit mono
)

// audiosocketGetStreamingID reads the first message from the audiosocket connection
// and extracts the streaming ID, which is expected to be a UUID.
//
// According to the Audiosocket protocol, the very first message sent on the connection
// must be a message of kind KindID, containing the streaming session's UUID.
//
// Parameters:
//
//	conn net.Conn - The network connection to read from.
//
// Returns:
//
//	uuid.UUID - The extracted streaming ID if successful, or uuid.Nil on error.
//	error     - Any error encountered during reading or validation.
//
// Errors are returned if:
//   - Reading the next message from the connection fails.
//   - The message kind is not KindID as expected.
//   - The payload cannot be parsed into a valid UUID.
func (h *streamingHandler) audiosocketGetStreamingID(conn net.Conn) (uuid.UUID, error) {
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
	if errWrite := buf.WriteByte(audiosocketFormatSLIN); errWrite != nil {
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

		fragmentLen := min(audiosocketMaxFragmentSize, payloadLen-offset)
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
		case <-time.After(audiosocketWriteDelay):
			// do nothing
			continue

		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
