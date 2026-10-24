package filehandler

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

// wavChunk encodes a RIFF chunk whose declared size may differ from the payload length.
func wavChunk(id string, declared uint32, payload []byte) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString(id)
	_ = binary.Write(buf, binary.LittleEndian, declared)
	buf.Write(payload)
	return buf.Bytes()
}

// wavFmtPayload encodes a 16-byte PCM fmt payload.
func wavFmtPayload(channels uint16, sampleRate uint32) []byte {
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, channels)
	_ = binary.Write(buf, binary.LittleEndian, sampleRate)
	_ = binary.Write(buf, binary.LittleEndian, sampleRate*uint32(channels)*2)
	_ = binary.Write(buf, binary.LittleEndian, channels*2)
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	return buf.Bytes()
}

func wavSamples(samples ...int16) []byte {
	buf := &bytes.Buffer{}
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

func wavFile(chunks ...[]byte) []byte {
	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))
	buf.WriteString("WAVE")
	for _, c := range chunks {
		buf.Write(c)
	}
	return buf.Bytes()
}

func Test_decodeWavPeaks_boundaries(t *testing.T) {
	fmtMono := wavChunk("fmt ", 16, wavFmtPayload(1, 8000))
	fmtStereo := wavChunk("fmt ", 16, wavFmtPayload(2, 8000))

	tests := []struct {
		name        string
		data        []byte
		bucketCount int
		expectPeaks []float64
		expectDur   float64
	}{
		{
			name:        "stereo frame peak is the larger channel magnitude",
			data:        wavFile(fmtStereo, wavChunk("data", 8, wavSamples(0, 16384, -32768, 0))),
			bucketCount: 2,
			expectPeaks: []float64{0.5, 1.0},
			expectDur:   0.00025,
		},
		{
			name:        "bucket count above frame count is clamped to frame count",
			data:        wavFile(fmtMono, wavChunk("data", 6, wavSamples(16384, -16384, 8192))),
			bucketCount: 10,
			expectPeaks: []float64{0.5, 0.5, 0.25},
			expectDur:   0.000375,
		},
		{
			name:        "header declares more frames than present and flushes the partial bucket",
			data:        wavFile(fmtMono, wavChunk("data", 8, wavSamples(16384, 8192, 4096))),
			bucketCount: 2,
			expectPeaks: []float64{0.5, 0.125},
			expectDur:   0.0005,
		},
		{
			name:        "declared data without any bytes keeps the duration and drops the peaks",
			data:        wavFile(fmtMono, wavChunk("data", 8, nil)),
			bucketCount: 2,
			expectPeaks: nil,
			expectDur:   0.0005,
		},
		{
			name:        "declared data size zero yields nothing",
			data:        wavFile(fmtMono, wavChunk("data", 0, nil)),
			bucketCount: 2,
			expectPeaks: nil,
			expectDur:   0,
		},
		{
			name:        "frames beyond the last full bucket are dropped",
			data:        wavFile(fmtMono, wavChunk("data", 10, wavSamples(8192, 4096, 16384, 2048, 24576))),
			bucketCount: 2,
			expectPeaks: []float64{0.25, 0.5},
			expectDur:   0.000625,
		},
		{
			name:        "odd declared data size drops the trailing byte",
			data:        wavFile(fmtMono, wavChunk("data", 5, append(wavSamples(16384, 8192), 0))),
			bucketCount: 2,
			expectPeaks: []float64{0.5, 0.25},
			expectDur:   0.00025,
		},
		{
			name: "unknown chunk with odd size is skipped with its pad byte",
			data: wavFile(
				wavChunk("LIST", 3, []byte{1, 2, 3, 0}),
				fmtMono,
				wavChunk("data", 4, wavSamples(16384, 8192)),
			),
			bucketCount: 2,
			expectPeaks: []float64{0.5, 0.25},
			expectDur:   0.00025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peaks, dur := decodeWavPeaks(bytes.NewReader(tt.data), tt.bucketCount)
			if !reflect.DeepEqual(peaks, tt.expectPeaks) {
				t.Errorf("Wrong match. expect peaks: %v, got: %v", tt.expectPeaks, peaks)
			}
			if math.Abs(dur-tt.expectDur) > 1e-12 {
				t.Errorf("Wrong match. expect duration: %v, got: %v", tt.expectDur, dur)
			}
		})
	}
}

func Test_readWavHeader_rejects(t *testing.T) {
	fmtMono := wavChunk("fmt ", 16, wavFmtPayload(1, 8000))

	tests := []struct {
		name string
		data []byte
	}{
		{"chunk header cut short", wavFile([]byte("fmt "))},
		{"fmt body cut short", wavFile(wavChunk("fmt ", 16, []byte{1, 0, 1, 0}))},
		{"fmt chunk smaller than 16 bytes", wavFile(wavChunk("fmt ", 8, make([]byte, 8)))},
		{"fmt chunk above the accepted maximum", wavFile(wavChunk("fmt ", 2048, nil))},
		{"data before fmt", wavFile(wavChunk("data", 4, wavSamples(1, 2)), fmtMono)},
		{"unknown chunk shorter than declared", wavFile(wavChunk("LIST", 10, []byte{1, 2}))},
		{"stream ends after fmt", wavFile(fmtMono)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, size, ok := readWavHeader(bytes.NewReader(tt.data))
			if ok || size != 0 {
				t.Errorf("Wrong match. expect: rejected, got ok: %v, size: %d", ok, size)
			}
		})
	}
}
