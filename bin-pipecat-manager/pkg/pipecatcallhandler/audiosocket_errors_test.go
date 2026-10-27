package pipecatcallhandler

import (
	"strings"
	"testing"
)

func Test_GetDataSamples_resamplerErrors(t *testing.T) {
	tests := []struct {
		name string

		inputRate int
		data      []byte

		expectErr string
	}{
		{
			name:      "one byte holds no whole sample",
			inputRate: 8000,
			data:      []byte{0x01},
			expectErr: "failed to write to resampler",
		},
		{
			name:      "negative input rate is rejected by the resampler",
			inputRate: -8000,
			data:      []byte{0x01},
			expectErr: "failed to create resampler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewAudiosocketHandler()

			res, err := h.GetDataSamples(tt.inputRate, tt.data)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong error. expect: %q, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}
