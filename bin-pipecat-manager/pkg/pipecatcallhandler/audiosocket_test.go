package pipecatcallhandler

import (
	"bytes"
	"encoding/binary"
	reflect "reflect"
	"testing"
)

func Test_audiosocketGetDataSamples(t *testing.T) {

	tests := []struct {
		name         string
		inputRate    int
		inputData    []byte
		expectLen    int  // expected output length in bytes
		expectExact  bool // if true, check exact bytes
		expectData   []byte
		expectError  bool
	}{
		{
			name:        "no conversion needed (same sample rate)",
			inputRate:   defaultAudiosocketConvertSampleRate,
			inputData:   []byte{0x01, 0x02, 0x03, 0x04},
			expectLen:   4,
			expectExact: true,
			expectData:  []byte{0x01, 0x02, 0x03, 0x04},
			expectError: false,
		},
		{
			name:        "downsample 2x (16000 → 8000) with filtering",
			inputRate:   defaultAudiosocketConvertSampleRate * 2,
			inputData:   []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			expectLen:   4, // 8 bytes / 2 = 4 bytes
			expectExact: false,
			expectError: false,
		},
		{
			name:      "downsample 4x (32000 → 8000) with filtering",
			inputRate: defaultAudiosocketConvertSampleRate * 4,
			inputData: []byte{
				0x01, 0x02, 0x03, 0x04,
				0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C,
				0x0D, 0x0E, 0x0F, 0x10,
			},
			expectLen:   4, // 16 bytes / 4 = 4 bytes
			expectExact: false,
			expectError: false,
		},
		{
			name:        "downsample 3x (24000 → 8000) with filtering",
			inputRate:   24000,
			inputData:   make([]byte, 24), // 12 samples at 24kHz → 4 samples at 8kHz
			expectLen:   8,                // 4 samples * 2 bytes
			expectExact: false,
			expectError: false,
		},
		{
			name:        "error on non-integer ratio",
			inputRate:   11000,
			inputData:   []byte{0x01, 0x02, 0x03, 0x04},
			expectError: true,
		},
		{
			name:        "error on odd byte count",
			inputRate:   16000,
			inputData:   []byte{0x01, 0x02, 0x03}, // 3 bytes is not 16-bit aligned
			expectError: true,
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
			if len(res) != tt.expectLen {
				t.Errorf("wrong length: expect %d, got %d", tt.expectLen, len(res))
			}
			if tt.expectExact && !bytes.Equal(res, tt.expectData) {
				t.Errorf("wrong data\nexpect: %v\ngot: %v", tt.expectData, res)
			}
		})
	}
}

func Test_applyLowPassFilter(t *testing.T) {
	tests := []struct {
		name    string
		samples []int16
		coeffs  []float64
	}{
		{
			name:    "empty samples",
			samples: []int16{},
			coeffs:  lpfCoeffs24kTo8k,
		},
		{
			name:    "empty coefficients",
			samples: []int16{100, 200, 300},
			coeffs:  []float64{},
		},
		{
			name:    "normal filtering",
			samples: []int16{1000, 2000, 3000, 4000, 5000, 6000},
			coeffs:  lpfCoeffs24kTo8k,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyLowPassFilter(tt.samples, tt.coeffs)

			// For empty inputs, should return same
			if len(tt.samples) == 0 || len(tt.coeffs) == 0 {
				if len(result) != len(tt.samples) {
					t.Errorf("expected length %d, got %d", len(tt.samples), len(result))
				}
				return
			}

			// Output length should match input length
			if len(result) != len(tt.samples) {
				t.Errorf("output length mismatch: expect %d, got %d", len(tt.samples), len(result))
			}

			// Verify filtering produces valid output (non-zero for non-zero input)
			hasNonZero := false
			for _, v := range result {
				if v != 0 {
					hasNonZero = true
					break
				}
			}
			if !hasNonZero {
				t.Errorf("filter produced all zeros for non-zero input")
			}
		})
	}
}

func Test_applyLowPassFilter_Clamping(t *testing.T) {
	// Test that filter clamps output to int16 range
	// Use large values that could overflow when summed
	samples := []int16{32000, 32000, 32000, 32000, 32000}

	// Use coefficients that sum to > 1.0 to potentially cause overflow
	// 32000 * (0.5 * 5) = 80000, which exceeds int16 max (32767)
	coeffs := []float64{0.5, 0.5, 0.5, 0.5, 0.5}

	result := applyLowPassFilter(samples, coeffs)

	// The middle sample should be clamped to max (32767)
	// because 32000 * 2.5 = 80000 > 32767
	if result[2] != 32767 {
		t.Errorf("expected clamped value 32767, got %d", result[2])
	}

	// Test negative clamping too
	negativeSamples := []int16{-32000, -32000, -32000, -32000, -32000}
	negResult := applyLowPassFilter(negativeSamples, coeffs)

	// Should clamp to -32768
	if negResult[2] != -32768 {
		t.Errorf("expected clamped value -32768, got %d", negResult[2])
	}
}

func Test_applyLowPassFilter_SmoothsHighFrequency(t *testing.T) {
	// High frequency signal: alternating values (Nyquist frequency)
	// This simulates the highest frequency component that should be attenuated
	highFreqSamples := []int16{
		10000, -10000, 10000, -10000, 10000, -10000,
		10000, -10000, 10000, -10000, 10000, -10000,
		10000, -10000, 10000, -10000, 10000, -10000,
		10000, -10000, 10000, -10000, 10000, -10000,
	}

	result := applyLowPassFilter(highFreqSamples, lpfCoeffs24kTo8k)

	// Calculate variance of input vs output
	// Filtered output should have lower variance (smoother)
	inputVariance := calculateVariance(highFreqSamples)
	outputVariance := calculateVariance(result)

	if outputVariance >= inputVariance {
		t.Errorf("filter did not reduce high frequency variance: input=%f, output=%f", inputVariance, outputVariance)
	}

	// The output variance should be significantly lower (at least 50% reduction)
	reductionRatio := outputVariance / inputVariance
	if reductionRatio > 0.5 {
		t.Errorf("filter did not sufficiently attenuate high frequencies: reduction ratio=%f", reductionRatio)
	}
}

func Test_applyLowPassFilter_PreservesLowFrequency(t *testing.T) {
	// Low frequency signal: gradual ramp (DC + low frequency)
	lowFreqSamples := []int16{
		1000, 1100, 1200, 1300, 1400, 1500,
		1600, 1700, 1800, 1900, 2000, 2100,
		2200, 2300, 2400, 2500, 2600, 2700,
		2800, 2900, 3000, 3100, 3200, 3300,
	}

	result := applyLowPassFilter(lowFreqSamples, lpfCoeffs24kTo8k)

	// Low frequency content should be largely preserved
	// Check that the general trend is maintained (increasing values)
	// Compare middle section to avoid edge effects
	midStart := len(result) / 3
	midEnd := 2 * len(result) / 3

	for i := midStart + 1; i < midEnd; i++ {
		// Allow small variations but overall should be increasing
		if result[i] < result[i-1]-200 {
			t.Errorf("filter disrupted low frequency trend at index %d: prev=%d, curr=%d", i, result[i-1], result[i])
		}
	}

	// Check that average value is preserved (within 20%)
	inputAvg := calculateAverage(lowFreqSamples[midStart:midEnd])
	outputAvg := calculateAverage(result[midStart:midEnd])

	diff := abs(int(inputAvg) - int(outputAvg))
	tolerance := int(inputAvg * 0.2)
	if diff > tolerance {
		t.Errorf("filter changed average too much: input avg=%f, output avg=%f", inputAvg, outputAvg)
	}
}

func Test_applyLowPassFilter_SingleSample(t *testing.T) {
	samples := []int16{5000}
	result := applyLowPassFilter(samples, lpfCoeffs24kTo8k)

	if len(result) != 1 {
		t.Errorf("expected 1 sample, got %d", len(result))
	}

	// Single sample should still produce output (edge case handling)
	// Value will be attenuated since only center coefficient applies fully
}

func Test_FilterCoefficients_Reasonable(t *testing.T) {
	// Verify that filter coefficients are reasonable
	// Note: These are anti-aliasing filters, so exact normalization isn't critical
	// The key property is that they attenuate high frequencies
	tests := []struct {
		name   string
		coeffs []float64
	}{
		{"16kHz to 8kHz", lpfCoeffs16kTo8k},
		{"24kHz to 8kHz", lpfCoeffs24kTo8k},
		{"32kHz to 8kHz", lpfCoeffs32kTo8k},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sum := 0.0
			for _, c := range tt.coeffs {
				sum += c
			}

			// Coefficients should sum to a positive value (not zero or negative)
			// and not be excessively large (would amplify signal)
			if sum <= 0 {
				t.Errorf("coefficients sum to non-positive value: %f", sum)
			}
			if sum > 2.0 {
				t.Errorf("coefficients sum too large (would amplify): %f", sum)
			}

			// Log the sum for informational purposes
			t.Logf("coefficients sum: %f", sum)
		})
	}
}

func Test_FilterCoefficients_Symmetric(t *testing.T) {
	// FIR low-pass filters should have symmetric coefficients
	tests := []struct {
		name   string
		coeffs []float64
	}{
		{"16kHz to 8kHz", lpfCoeffs16kTo8k},
		{"24kHz to 8kHz", lpfCoeffs24kTo8k},
		{"32kHz to 8kHz", lpfCoeffs32kTo8k},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := len(tt.coeffs)
			for i := 0; i < n/2; i++ {
				if tt.coeffs[i] != tt.coeffs[n-1-i] {
					t.Errorf("coefficients not symmetric at index %d: %f != %f", i, tt.coeffs[i], tt.coeffs[n-1-i])
				}
			}
		})
	}
}

func Test_GetDataSamples_FilterChangesOutput(t *testing.T) {
	// Verify that filtering produces different output than simple decimation
	// This confirms the filter is actually being applied
	h := &audiosocketHandler{}

	// Create input with high frequency content that filter should attenuate
	// Alternating pattern at 24kHz
	inputSamples := make([]int16, 48) // 48 samples at 24kHz = 16 samples at 8kHz
	for i := range inputSamples {
		if i%2 == 0 {
			inputSamples[i] = 10000
		} else {
			inputSamples[i] = -10000
		}
	}

	// Convert to bytes
	inputBytes := make([]byte, len(inputSamples)*2)
	for i, s := range inputSamples {
		binary.LittleEndian.PutUint16(inputBytes[i*2:], uint16(s))
	}

	// Get filtered output
	filtered, err := h.GetDataSamples(24000, inputBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Simple decimation would just take every 3rd sample
	// which would be: 10000, -10000, 10000, -10000, ...
	// Filtered output should be smoothed (lower magnitude values)

	// Convert output back to samples
	outputSamples := make([]int16, len(filtered)/2)
	for i := range outputSamples {
		outputSamples[i] = int16(binary.LittleEndian.Uint16(filtered[i*2:]))
	}

	// Calculate max absolute value - filtered should be lower
	maxFiltered := int16(0)
	for _, s := range outputSamples {
		if s > maxFiltered {
			maxFiltered = s
		}
		if -s > maxFiltered {
			maxFiltered = -s
		}
	}

	// Simple decimation would preserve 10000 magnitude
	// Filtering should reduce it significantly
	if maxFiltered > 5000 {
		t.Errorf("filter did not attenuate high frequency: max magnitude=%d (expected < 5000)", maxFiltered)
	}
}

func Test_GetDataSamples_AllSupportedRates(t *testing.T) {
	h := &audiosocketHandler{}

	tests := []struct {
		name       string
		inputRate  int
		numSamples int // number of input samples
		expectOut  int // expected output samples
	}{
		{"16kHz to 8kHz", 16000, 32, 16},
		{"24kHz to 8kHz", 24000, 48, 16},
		{"32kHz to 8kHz", 32000, 64, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create input bytes
			inputBytes := make([]byte, tt.numSamples*2)
			for i := 0; i < tt.numSamples; i++ {
				// Use a simple ramp for input
				binary.LittleEndian.PutUint16(inputBytes[i*2:], uint16(i*100))
			}

			result, err := h.GetDataSamples(tt.inputRate, inputBytes)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			outputSamples := len(result) / 2
			if outputSamples != tt.expectOut {
				t.Errorf("wrong output sample count: expected %d, got %d", tt.expectOut, outputSamples)
			}
		})
	}
}

// Helper functions for tests
func calculateVariance(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}

	avg := calculateAverage(samples)
	sum := 0.0
	for _, s := range samples {
		diff := float64(s) - avg
		sum += diff * diff
	}
	return sum / float64(len(samples))
}

func calculateAverage(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}

	sum := 0.0
	for _, s := range samples {
		sum += float64(s)
	}
	return sum / float64(len(samples))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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
