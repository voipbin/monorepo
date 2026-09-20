package filehandler

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/pkg/errors"

	"monorepo/bin-storage-manager/models/file"
)

// peakBucketCount is the fixed number of downsampled waveform buckets returned
// per file. A fixed count keeps the response size and frontend rendering cost
// constant regardless of recording length.
const peakBucketCount = 1000

// wavFormat holds the subset of WAV header fields needed to decode PCM samples.
type wavFormat struct {
	audioFormat   uint16
	numChannels   uint16
	sampleRate    uint32
	bitsPerSample uint16
}

// PeaksComputeByFile reads the given file's WAV bytes from the bucket and
// returns the normalized (-1..1) downsampled waveform peaks and the audio
// duration in seconds. It never fails the caller for a bad file: on any decode
// error it returns empty peaks and 0 duration so playback (which does not need
// peaks) is unaffected.
func (h *fileHandler) PeaksComputeByFile(ctx context.Context, f *file.File, bucketCount int) ([]float64, float64, error) {
	if bucketCount <= 0 {
		bucketCount = peakBucketCount
	}

	reader, err := h.client.Bucket(f.BucketName).Object(f.Filepath).NewReader(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "could not open the file reader. bucket_name: %s, filepath: %s", f.BucketName, f.Filepath)
	}
	defer func() {
		_ = reader.Close()
	}()

	peaks, duration := decodeWavPeaks(reader, bucketCount)
	return peaks, duration, nil
}

// decodeWavPeaks parses a PCM WAV stream and produces the peak envelope and
// duration. Any parse failure yields (nil, 0) rather than an error: a missing
// waveform must not break playback.
func decodeWavPeaks(r io.Reader, bucketCount int) ([]float64, float64) {
	format, dataSize, ok := readWavHeader(r)
	if !ok || format.bitsPerSample != 16 || format.audioFormat != 1 || format.numChannels == 0 {
		// Only linear 16-bit PCM is supported; anything else yields no waveform.
		return nil, 0
	}

	bytesPerSample := int(format.bitsPerSample / 8)
	frameSize := bytesPerSample * int(format.numChannels)
	if frameSize == 0 {
		return nil, 0
	}

	totalFrames := int(dataSize) / frameSize
	if totalFrames <= 0 {
		return nil, 0
	}

	duration := float64(totalFrames) / float64(format.sampleRate)

	if bucketCount > totalFrames {
		bucketCount = totalFrames
	}
	framesPerBucket := totalFrames / bucketCount
	if framesPerBucket <= 0 {
		framesPerBucket = 1
	}

	peaks := make([]float64, 0, bucketCount)

	buf := make([]byte, frameSize)
	frameIdx := 0
	var bucketMax float64
	bucketFrameCount := 0

	for frameIdx < totalFrames {
		if _, err := io.ReadFull(r, buf); err != nil {
			break
		}

		// Peak across channels for this frame (mono downmix by max magnitude).
		var frameMax float64
		for ch := 0; ch < int(format.numChannels); ch++ {
			off := ch * bytesPerSample
			sample := int16(binary.LittleEndian.Uint16(buf[off : off+2]))
			mag := float64(sample) / 32768.0
			if mag < 0 {
				mag = -mag
			}
			if mag > frameMax {
				frameMax = mag
			}
		}
		if frameMax > bucketMax {
			bucketMax = frameMax
		}

		bucketFrameCount++
		frameIdx++

		if bucketFrameCount >= framesPerBucket && len(peaks) < bucketCount {
			peaks = append(peaks, bucketMax)
			bucketMax = 0
			bucketFrameCount = 0
		}
	}

	// Flush a trailing partial bucket if room remains.
	if bucketFrameCount > 0 && len(peaks) < bucketCount {
		peaks = append(peaks, bucketMax)
	}

	if len(peaks) == 0 {
		return nil, duration
	}
	return peaks, duration
}

// readWavHeader scans RIFF chunks for `fmt ` and `data`, returning the audio
// format and the data chunk's byte size. The reader is left positioned at the
// start of the PCM data on success.
func readWavHeader(r io.Reader) (wavFormat, uint32, bool) {
	var format wavFormat

	// RIFF header: "RIFF" <size u32> "WAVE"
	riff := make([]byte, 12)
	if _, err := io.ReadFull(r, riff); err != nil {
		return format, 0, false
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return format, 0, false
	}

	gotFmt := false
	chunkHeader := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, chunkHeader); err != nil {
			return format, 0, false
		}
		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			fmtBuf := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtBuf); err != nil {
				return format, 0, false
			}
			if chunkSize < 16 {
				return format, 0, false
			}
			format.audioFormat = binary.LittleEndian.Uint16(fmtBuf[0:2])
			format.numChannels = binary.LittleEndian.Uint16(fmtBuf[2:4])
			format.sampleRate = binary.LittleEndian.Uint32(fmtBuf[4:8])
			format.bitsPerSample = binary.LittleEndian.Uint16(fmtBuf[14:16])
			gotFmt = true
		case "data":
			if !gotFmt {
				return format, 0, false
			}
			return format, chunkSize, true
		default:
			// Skip unknown chunk (word-aligned).
			skip := int64(chunkSize)
			if skip%2 == 1 {
				skip++
			}
			if _, err := io.CopyN(io.Discard, r, skip); err != nil {
				return format, 0, false
			}
		}
	}
}
