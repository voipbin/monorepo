package streaminghandler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

const (
	audiosocketFormatSLIN uint16 = 0x0010 // SLIN format for 16-bit PCM audio
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
//   - 2 bytes: Audio format identifier (uint16, BigEndian), fixed to audiosocketFormatSLIN (0x0010 for signed linear PCM)
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
		return nil, fmt.Errorf("PCM data must be 16-bit aligned (even number of bytes)")
	}

	sampleCount := len(data) / 2 // 2 bytes per sample (16-bit)

	buf := new(bytes.Buffer)

	// Write audio format (SLIN)
	if errWrite := binary.Write(buf, binary.BigEndian, audiosocketFormatSLIN); errWrite != nil {
		return nil, errors.Wrapf(errWrite, "could not write audio format")
	}

	// Write sample count
	if errWrite := binary.Write(buf, binary.BigEndian, uint16(sampleCount)); errWrite != nil {
		return nil, errors.Wrapf(errWrite, "could not write sample count")
	}

	// Write raw PCM data
	_, err := buf.Write(data)
	if err != nil {
		return nil, errors.Wrapf(err, "could not write raw audio data")
	}

	return buf.Bytes(), nil
}
