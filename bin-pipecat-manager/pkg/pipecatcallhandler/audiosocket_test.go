package pipecatcallhandler

import (
	"bytes"
	"encoding/binary"
	reflect "reflect"
	"testing"
)

func Test_audiosocketGetDataSamples(t *testing.T) {

	tests := []struct {
		name             string
		inputRate        int
		inputSamples     int // number of 16-bit samples
		expectExactMatch bool
		expectError      bool
	}{
		{
			name:             "no conversion needed (same sample rate)",
			inputRate:        defaultAudiosocketConvertSampleRate,
			inputSamples:     100,
			expectExactMatch: true,
			expectError:      false,
		},
		{
			name:             "downsample 2x (16000 → 8000)",
			inputRate:        defaultAudiosocketConvertSampleRate * 2,
			inputSamples:     100,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "downsample 3x (24000 → 8000)",
			inputRate:        defaultAudiosocketConvertSampleRate * 3,
			inputSamples:     120,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "downsample 4x (32000 → 8000)",
			inputRate:        defaultAudiosocketConvertSampleRate * 4,
			inputSamples:     200,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "empty input",
			inputRate:        defaultAudiosocketConvertSampleRate * 2,
			inputSamples:     0,
			expectExactMatch: true,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &audiosocketHandler{}

			// Generate test PCM data (sine wave pattern)
			inputData := make([]byte, tt.inputSamples*2)
			for i := 0; i < tt.inputSamples; i++ {
				sample := int16(1000 * i / (tt.inputSamples + 1)) // Simple ramp
				binary.LittleEndian.PutUint16(inputData[i*2:], uint16(sample))
			}

			res, err := h.GetDataSamples(tt.inputRate, inputData)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectExactMatch {
				if !bytes.Equal(res, inputData) {
					t.Errorf("expected exact match but got different data")
				}
				return
			}

			// For resampled data, verify approximate output size
			expectedSamples := tt.inputSamples * defaultAudiosocketConvertSampleRate / tt.inputRate
			actualSamples := len(res) / 2

			// Allow 10% margin for resampling variations
			minSamples := expectedSamples * 9 / 10
			maxSamples := expectedSamples * 11 / 10

			if actualSamples < minSamples || actualSamples > maxSamples {
				t.Errorf("output sample count out of range: expected ~%d, got %d (range: %d-%d)",
					expectedSamples, actualSamples, minSamples, maxSamples)
			}

			// Verify output is valid (even number of bytes)
			if len(res)%2 != 0 {
				t.Errorf("output length must be even (16-bit aligned), got %d", len(res))
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

			out, err := h.Upsample8kTo16k(inputBytes.Bytes())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

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
