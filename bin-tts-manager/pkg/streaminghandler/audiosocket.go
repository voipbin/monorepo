package streaminghandler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/CyCoreSystems/audiosocket"
	"github.com/gofrs/uuid"
)

// audiosocketGetStreamingID gets the streaming id from the connection
// the first message of the audiosocket should be the streaming id
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

// audiosocketWrapData wraps the raw audio data into a format suitable for audiosocket transmission.
// [2 bytes: audio format]
// [2 bytes: sample count]
// [raw audio payload (PCM, μ-law, etc.)]
func audiosocketWrapData(data []byte) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("PCM data must be 16-bit aligned (even number of bytes)")
	}

	sampleCount := len(data) / 2 // 2 bytes per sample (16-bit)

	buf := new(bytes.Buffer)

	// Write audio format (0x0010 for SLIN)
	_ = binary.Write(buf, binary.BigEndian, uint16(0x0010))

	// Write sample count
	_ = binary.Write(buf, binary.BigEndian, uint16(sampleCount))

	// Write raw PCM data
	_, err := buf.Write(data)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
