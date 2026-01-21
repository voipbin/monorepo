# Audio Resampling Design

## Problem

AI voice quality is degraded (choppy, garbled) due to improper downsampling. ElevenLabs TTS outputs 24kHz audio, but Audiosocket requires 8kHz. The current implementation uses simple decimation (taking every 3rd sample), which causes aliasing artifacts because high-frequency content above 4kHz folds back into the audible range.

## Solution

Use `zaf/resample`, a Go binding to libsoxr (SoX Resampler Library), for professional-quality audio resampling with proper anti-aliasing filters.

## Integration Point

Replace the simple decimation logic in `pkg/pipecatcallhandler/audiosocket.go`:

**Current location:** `GetDataSamples()` function at line 79

**Current behavior:** Takes every Nth sample without filtering

**New behavior:** Use libsoxr resampling with anti-aliasing

## Implementation

```go
import "github.com/zaf/resample"

func (h *audiosocketHandler) GetDataSamples(inputRate int, data []byte) ([]byte, error) {
    if inputRate == defaultAudiosocketConvertSampleRate {
        return data, nil
    }

    if len(data) == 0 {
        return data, nil
    }

    // Convert bytes to float64 samples (libsoxr works with float64)
    numSamples := len(data) / 2
    floatSamples := make([]float64, numSamples)
    for i := 0; i < numSamples; i++ {
        sample := int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
        floatSamples[i] = float64(sample) / 32768.0
    }

    // Create resampler: input rate -> 8kHz, mono channel, MediumQ quality
    resampler, err := resample.New(
        floatSamples,
        float64(inputRate),
        float64(defaultAudiosocketConvertSampleRate),
        1,                    // mono
        resample.MediumQ,     // balance quality vs CPU
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resampler: %w", err)
    }

    // Read all resampled output
    outputSamples := make([]float64, numSamples*defaultAudiosocketConvertSampleRate/inputRate+10)
    n, err := resampler.Read(outputSamples)
    if err != nil && err != io.EOF {
        return nil, fmt.Errorf("failed to resample: %w", err)
    }

    // Convert back to 16-bit PCM bytes
    result := make([]byte, n*2)
    for i := 0; i < n; i++ {
        sample := int16(outputSamples[i] * 32767.0)
        binary.LittleEndian.PutUint16(result[i*2:], uint16(sample))
    }

    return result, nil
}
```

## Docker Requirements

Update `Dockerfile` to install libsoxr:

```dockerfile
# Build stage
FROM golang:1.21-bullseye AS builder
RUN apt-get update && apt-get install -y libsoxr-dev pkg-config

# Runtime stage
FROM debian:bullseye-slim
RUN apt-get update && apt-get install -y libsoxr0
```

## Go Module

Add dependency:

```bash
go get github.com/zaf/resample
```

## Quality Setting

Use `resample.MediumQ` as the default:
- `LowQ`: Fastest, acceptable for voice
- `MediumQ`: Good balance (recommended)
- `HighQ`: Highest quality, more CPU
- `VeryHighQ`: Maximum quality, highest CPU

For real-time voice, MediumQ provides good quality without excessive latency.

## Error Handling

- Return original data if resampling fails (graceful degradation)
- Log warnings for resampling errors
- Handle empty input gracefully

## Testing

1. Unit tests with known audio patterns
2. Test sample rate conversions: 16kHz, 24kHz, 32kHz, 48kHz -> 8kHz
3. Verify no aliasing artifacts in output
4. Measure CPU usage under load

## Rollback Plan

If issues arise, revert to simple decimation by removing the resample import and restoring the original loop-based downsampling.
