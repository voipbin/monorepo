package streaminghandler

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func Test_audiosocketWrapData16BitsPCM(t *testing.T) {
	type test struct {
		name       string
		input      []byte
		expectData []byte
		expectErr  bool
	}

	tests := []test{
		{
			name:  "valid 16-bit PCM data (2 samples)",
			input: []byte{0x01, 0x02, 0x03, 0x04}, // 2 samples
			expectData: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.BigEndian, audiosocketFormatSLIN) // audio format
				_ = binary.Write(buf, binary.BigEndian, uint16(2))             // sample count
				_, _ = buf.Write([]byte{0x01, 0x02, 0x03, 0x04})               // raw PCM
				return buf.Bytes()
			}(),
			expectErr: false,
		},
		{
			name:      "invalid: odd-length input",
			input:     []byte{0x01, 0x02, 0x03}, // not 16-bit aligned
			expectErr: true,
		},
		{
			name:  "empty input (0 samples)",
			input: []byte{},
			expectData: func() []byte {
				buf := new(bytes.Buffer)
				_ = binary.Write(buf, binary.BigEndian, audiosocketFormatSLIN)
				_ = binary.Write(buf, binary.BigEndian, uint16(0))
				return buf.Bytes()
			}(),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := audiosocketWrapDataPCM16Bit(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !bytes.Equal(output, tt.expectData) {
				t.Errorf("output mismatch.\nexpected: %v\ngot:      %v", tt.expectData, output)
			}
		})
	}
}
