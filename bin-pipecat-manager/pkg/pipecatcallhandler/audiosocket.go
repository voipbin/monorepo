package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_audiosocket.go -source audiosocket.go -build_flags=-mod=mod

import (
	"fmt"

	"github.com/zaf/resample"
)

type AudiosocketHandler interface {
	GetDataSamples(inputRate int, data []byte) ([]byte, error)
}

const (
	defaultConvertSampleRate = 16000 // Target sample rate for resampling (16kHz slin16)
)

type audiosocketHandler struct{}

func NewAudiosocketHandler() AudiosocketHandler {
	return &audiosocketHandler{}
}

// GetDataSamples resamples 16-bit PCM data from inputRate to 16kHz.
// If inputRate already equals 16kHz, it returns data unchanged.
func (h *audiosocketHandler) GetDataSamples(inputRate int, data []byte) ([]byte, error) {
	if inputRate == defaultConvertSampleRate {
		// No conversion needed
		return data, nil
	}

	if len(data) == 0 {
		return data, nil
	}

	// Get buffer from pool and estimate output size
	inputSamples := len(data) / 2
	outputSamples := inputSamples * defaultConvertSampleRate / inputRate
	output := getBuffer()
	output.Grow(outputSamples * 2)

	// Create resampler: input rate -> 16kHz, mono channel, I16 format, MediumQ quality
	resampler, err := resample.New(
		output,
		float64(inputRate),
		float64(defaultConvertSampleRate),
		1,                // mono
		resample.I16,     // 16-bit signed linear PCM
		resample.MediumQ, // balance quality vs CPU
	)
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to create resampler: %w", err)
	}

	// Write input data to the resampler
	_, err = resampler.Write(data)
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to write to resampler: %w", err)
	}

	// Close to flush any remaining output
	err = resampler.Close()
	if err != nil {
		putBuffer(output)
		return nil, fmt.Errorf("failed to close resampler: %w", err)
	}

	// Copy result before returning buffer to pool
	result := make([]byte, output.Len())
	copy(result, output.Bytes())
	putBuffer(output)

	return result, nil
}
