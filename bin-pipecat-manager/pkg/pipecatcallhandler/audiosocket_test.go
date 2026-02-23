package pipecatcallhandler

import (
	"bytes"
	"encoding/binary"
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
			inputRate:        defaultConvertSampleRate,
			inputSamples:     100,
			expectExactMatch: true,
			expectError:      false,
		},
		{
			name:             "downsample 2x (32000 → 16000)",
			inputRate:        defaultConvertSampleRate * 2,
			inputSamples:     100,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "downsample 3x (48000 → 16000)",
			inputRate:        defaultConvertSampleRate * 3,
			inputSamples:     120,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "downsample 4x (64000 → 16000)",
			inputRate:        defaultConvertSampleRate * 4,
			inputSamples:     200,
			expectExactMatch: false,
			expectError:      false,
		},
		{
			name:             "empty input",
			inputRate:        defaultConvertSampleRate * 2,
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
			expectedSamples := tt.inputSamples * defaultConvertSampleRate / tt.inputRate
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
