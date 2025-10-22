package pipecatcallhandler

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func Test_audiosocketGetDataSamples(t *testing.T) {
	type test struct {
		name        string
		inputRate   int
		inputData   []byte
		expectData  []byte
		expectError bool
	}

	tests := []test{
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
			res, err := audiosocketGetDataSamples(tt.inputRate, tt.inputData)
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
	type test struct {
		name       string
		inputData  []int16
		expectData []int16
	}

	tests := []test{
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
			var inputBytes bytes.Buffer
			for _, v := range tt.inputData {
				_ = binary.Write(&inputBytes, binary.LittleEndian, v)
			}

			out := audiosocketUpsample8kTo16k(inputBytes.Bytes())

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
	type test struct {
		name        string
		inputData   []byte
		expectError bool
	}

	tests := []test{
		{
			name:        "normal case even length",
			inputData:   []byte{0x01, 0x02, 0x03, 0x04}, // 2 샘플
			expectError: false,
		},
		{
			name:        "error case odd length",
			inputData:   []byte{0x01, 0x02, 0x03}, // 1.5 샘플 → error
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped, err := audiosocketWrapDataPCM16Bit(tt.inputData)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// 최소 길이 체크: 1 byte format + 2 byte sample count + payload
			if len(wrapped) < 3 {
				t.Fatalf("wrapped data too short: %d", len(wrapped))
			}

			// 첫 바이트: format
			if wrapped[0] != defaultAudiosocketFormatSLIN {
				t.Errorf("wrong format byte: expect 0x%x, got 0x%x", defaultAudiosocketFormatSLIN, wrapped[0])
			}

			// 다음 2 바이트: sample count
			sampleCount := binary.BigEndian.Uint16(wrapped[1:3])
			expectedSampleCount := uint16(len(tt.inputData) / 2)
			if sampleCount != expectedSampleCount {
				t.Errorf("wrong sample count: expect %d, got %d", expectedSampleCount, sampleCount)
			}

			// 나머지 payload: 입력과 동일
			if !bytes.Equal(tt.inputData, wrapped[3:]) {
				t.Errorf("payload mismatch: expect %v, got %v", tt.inputData, wrapped[3:])
			}
		})
	}
}
