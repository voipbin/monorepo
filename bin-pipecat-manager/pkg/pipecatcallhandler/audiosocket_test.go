package pipecatcallhandler

import (
	"bytes"
	"encoding/binary"
	reflect "reflect"
	"testing"
)

func Test_audiosocketGetDataSamples(t *testing.T) {

	tests := []struct {
		name        string
		inputRate   int
		inputData   []byte
		expectData  []byte
		expectError bool
	}{
		{
			name:        "no conversion needed (same sample rate)",
			inputRate:   defaultAudiosocketConvertSampleRate,
			inputData:   []byte{0x01, 0x02, 0x03, 0x04},
			expectData:  []byte{0x01, 0x02, 0x03, 0x04},
			expectError: false,
		},
		{
			name:        "downsample 2x (16000 → 8000)",
			inputRate:   defaultAudiosocketConvertSampleRate * 2,
			inputData:   []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			expectData:  []byte{0x11, 0x22, 0x55, 0x66},
			expectError: false,
		},
		{
			name:      "downsample 4x (32000 → 8000)",
			inputRate: defaultAudiosocketConvertSampleRate * 4,
			inputData: []byte{
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C,
				0x0D, 0x0E, 0x0F, 0x10,
			},
			expectData:  []byte{0x01, 0x02, 0x09, 0x0A},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &audiosocketHandler{}

			res, err := h.GetDataSamples(tt.inputRate, tt.inputData)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !bytes.Equal(res, tt.expectData) {
				t.Errorf("wrong data\nexpect: %v\ngot: %v", tt.expectData, res)
			}
		})
	}
}

func Test_audiosocketUpsample8kTo16k(t *testing.T) {

	tests := []struct {
		name       string
		inputData  []int16
		expectData []int16
	}{
		{
			name:      "normal upsample",
			inputData: []int16{1000, 2000, 3000, 4000},
			expectData: []int16{
				1000, 1500, 2000, 2500, 3000, 3500, 4000,
			},
		},
		{
			name:       "empty input",
			inputData:  []int16{},
			expectData: []int16{},
		},
		{
			name:      "odd length input",
			inputData: []int16{1000, 2000, 3000},
			expectData: []int16{
				1000, 1500, 2000, 2500, 3000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &audiosocketHandler{}

			var inputBytes bytes.Buffer
			for _, v := range tt.inputData {
				_ = binary.Write(&inputBytes, binary.LittleEndian, v)
			}

			out := h.Upsample8kTo16k(inputBytes.Bytes())

			outSamples := make([]int16, len(out)/2)
			for i := 0; i < len(outSamples); i++ {
				outSamples[i] = int16(binary.LittleEndian.Uint16(out[i*2 : i*2+2]))
			}

			if len(outSamples) != len(tt.expectData) {
				t.Fatalf("length mismatch: expect %d, got %d", len(tt.expectData), len(outSamples))
			}

			for i := range outSamples {
				if outSamples[i] != tt.expectData[i] {
					t.Errorf("index %d: expect %d, got %d", i, tt.expectData[i], outSamples[i])
				}
			}
		})
	}
}

func Test_audiosocketWrapDataPCM16Bit(t *testing.T) {
	tests := []struct {
		name      string
		inputData []int16
		expectRes []byte
	}{
		{
			name:      "normal pcm data",
			inputData: []int16{1000, 2000},
			// 0x10                : format byte (defaultAudiosocketFormatSLIN)
			// 0x00, 0x04          : sample count (BigEndian, 2)
			// 0xE8, 0x03, 0xD0, 0x07 : PCM16 LE(1000, 2000)
			expectRes: []byte{0x10, 0x00, 0x04, 0xE8, 0x03, 0xD0, 0x07},
		},
		{
			name:      "empty pcm data",
			inputData: []int16{},
			// 0x10 : format
			// 0x00, 0x01 : sample count = 1
			// 0xD2, 0x04 : PCM16 LE(1234)
			expectRes: []byte{0x10, 0x00, 0x00},
		},
		{
			name:      "single sample",
			inputData: []int16{1234},
			// 0x10 : format
			// 0x00, 0x01 : sample count = 1
			// 0xD2, 0x04 : PCM16 LE(1234)
			expectRes: []byte{0x10, 0x00, 0x02, 0xD2, 0x04},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &audiosocketHandler{}

			var inputBuf bytes.Buffer
			for _, v := range tt.inputData {
				_ = binary.Write(&inputBuf, binary.LittleEndian, v)
			}

			out, err := h.WrapDataPCM16Bit(inputBuf.Bytes())
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(out, tt.expectRes) {
				t.Errorf("output mismatch\nexpect: %v\ngot:    %v", tt.expectRes, out)
			}
		})
	}
}
