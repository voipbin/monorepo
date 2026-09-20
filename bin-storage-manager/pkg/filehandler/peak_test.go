package filehandler

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"
)

// buildTestWav builds a minimal 16-bit PCM mono WAV in memory from the given samples.
func buildTestWav(sampleRate uint32, samples []int16) []byte {
	buf := &bytes.Buffer{}

	dataSize := uint32(len(samples) * 2)
	byteRate := sampleRate * 2

	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // fmt chunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // audioFormat PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // numChannels
	_ = binary.Write(buf, binary.LittleEndian, sampleRate)
	_ = binary.Write(buf, binary.LittleEndian, byteRate)
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))  // block align
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // bits per sample

	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, dataSize)
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes()
}

func Test_decodeWavPeaks(t *testing.T) {
	tests := []struct {
		name         string
		sampleRate   uint32
		samples      []int16
		bucketCount  int
		expectDur    float64
		expectPeaks  int
		expectMaxOne bool // whether at least one peak should be ~1.0 (full-scale sample present)
	}{
		{
			name:         "full scale samples produce normalized peaks near 1",
			sampleRate:   8000,
			samples:      []int16{0, 16384, -32768, 32767, 0, -16384, 8000, -8000},
			bucketCount:  4,
			expectDur:    8.0 / 8000.0,
			expectPeaks:  4,
			expectMaxOne: true,
		},
		{
			name:         "single bucket",
			sampleRate:   8000,
			samples:      []int16{100, 200, 300, 400},
			bucketCount:  1,
			expectDur:    4.0 / 8000.0,
			expectPeaks:  1,
			expectMaxOne: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wav := buildTestWav(tt.sampleRate, tt.samples)
			peaks, dur := decodeWavPeaks(bytes.NewReader(wav), tt.bucketCount)

			if len(peaks) != tt.expectPeaks {
				t.Errorf("wrong peak count. expect: %d, got: %d", tt.expectPeaks, len(peaks))
			}
			if math.Abs(dur-tt.expectDur) > 1e-9 {
				t.Errorf("wrong duration. expect: %f, got: %f", tt.expectDur, dur)
			}
			for _, p := range peaks {
				if p < 0 || p > 1.0 {
					t.Errorf("peak out of normalized range [0,1]. got: %f", p)
				}
			}
			if tt.expectMaxOne {
				var maxP float64
				for _, p := range peaks {
					if p > maxP {
						maxP = p
					}
				}
				if maxP < 0.99 {
					t.Errorf("expected a near-full-scale peak. got max: %f", maxP)
				}
			}
		})
	}
}

func Test_decodeWavPeaks_invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"not riff", []byte("XXXXsomethingnotwav")},
		{"truncated header", []byte("RIFF")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peaks, dur := decodeWavPeaks(bytes.NewReader(tt.data), 100)
			if peaks != nil {
				t.Errorf("expected nil peaks for invalid input. got: %v", peaks)
			}
			if dur != 0 {
				t.Errorf("expected 0 duration for invalid input. got: %f", dur)
			}
		})
	}
}

// Test_decodeWavPeaks_zeroSampleRate guards against a WAV header declaring a
// zero sample rate, which would otherwise make duration +Inf and break JSON
// marshaling of the peaks response.
func Test_decodeWavPeaks_zeroSampleRate(t *testing.T) {
	wav := buildTestWav(0, []int16{100, -100, 200, -200})

	peaks, dur := decodeWavPeaks(bytes.NewReader(wav), 100)
	if peaks != nil {
		t.Errorf("expected nil peaks for zero sample rate. got: %v", peaks)
	}
	if dur != 0 {
		t.Errorf("expected 0 duration for zero sample rate (not +Inf). got: %f", dur)
	}
}
