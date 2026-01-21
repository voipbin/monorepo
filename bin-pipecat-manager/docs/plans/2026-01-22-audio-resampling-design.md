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

The `zaf/resample` library uses an `io.WriteCloser` pattern with native PCM format support:

```go
import "github.com/zaf/resample"

func (h *audiosocketHandler) GetDataSamples(inputRate int, data []byte) ([]byte, error) {
    if inputRate == defaultAudiosocketConvertSampleRate {
        return data, nil
    }

    if len(data) == 0 {
        return data, nil
    }

    // Create output buffer
    var output bytes.Buffer

    // Create resampler: input rate -> 8kHz, mono channel, I16 format, MediumQ quality
    resampler, err := resample.New(
        &output,
        float64(inputRate),
        float64(defaultAudiosocketConvertSampleRate),
        1,                // mono
        resample.I16,     // 16-bit signed linear PCM (native format)
        resample.MediumQ, // balance quality vs CPU
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create resampler: %w", err)
    }

    // Write input data to the resampler
    _, err = resampler.Write(data)
    if err != nil {
        return nil, fmt.Errorf("failed to write to resampler: %w", err)
    }

    // Close to flush any remaining output
    err = resampler.Close()
    if err != nil {
        return nil, fmt.Errorf("failed to close resampler: %w", err)
    }

    return output.Bytes(), nil
}
```

**API Notes:**
- `resample.New(writer, inputRate, outputRate, channels, format, quality)`
- Format options: `I16` (16-bit PCM), `I32` (32-bit), `F32` (float32), `F64` (float64)
- Using `I16` allows direct pass-through of PCM bytes without conversion

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
