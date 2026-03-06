package ai

import (
	"testing"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func TestVADConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		config    *VADConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil config is valid",
			config:    nil,
			wantError: false,
		},
		{
			name:      "empty config is valid",
			config:    &VADConfig{},
			wantError: false,
		},
		{
			name: "all valid values",
			config: &VADConfig{
				Confidence: float64Ptr(0.7),
				StartSecs:  float64Ptr(0.2),
				StopSecs:   float64Ptr(0.5),
				MinVolume:  float64Ptr(0.6),
			},
			wantError: false,
		},
		{
			name: "zero values are valid",
			config: &VADConfig{
				Confidence: float64Ptr(0.0),
				StartSecs:  float64Ptr(0.0),
				StopSecs:   float64Ptr(0.0),
				MinVolume:  float64Ptr(0.0),
			},
			wantError: false,
		},
		{
			name: "boundary values 1.0 are valid",
			config: &VADConfig{
				Confidence: float64Ptr(1.0),
				MinVolume:  float64Ptr(1.0),
			},
			wantError: false,
		},
		{
			name: "confidence above 1.0 is invalid",
			config: &VADConfig{
				Confidence: float64Ptr(1.1),
			},
			wantError: true,
			errorMsg:  "confidence must be between 0.0 and 1.0",
		},
		{
			name: "confidence below 0.0 is invalid",
			config: &VADConfig{
				Confidence: float64Ptr(-0.1),
			},
			wantError: true,
			errorMsg:  "confidence must be between 0.0 and 1.0",
		},
		{
			name: "min_volume above 1.0 is invalid",
			config: &VADConfig{
				MinVolume: float64Ptr(1.5),
			},
			wantError: true,
			errorMsg:  "min_volume must be between 0.0 and 1.0",
		},
		{
			name: "min_volume below 0.0 is invalid",
			config: &VADConfig{
				MinVolume: float64Ptr(-0.5),
			},
			wantError: true,
			errorMsg:  "min_volume must be between 0.0 and 1.0",
		},
		{
			name: "negative start_secs is invalid",
			config: &VADConfig{
				StartSecs: float64Ptr(-1.0),
			},
			wantError: true,
			errorMsg:  "start_secs must be non-negative",
		},
		{
			name: "negative stop_secs is invalid",
			config: &VADConfig{
				StopSecs: float64Ptr(-0.01),
			},
			wantError: true,
			errorMsg:  "stop_secs must be non-negative",
		},
		{
			name: "partial config with only stop_secs",
			config: &VADConfig{
				StopSecs: float64Ptr(0.8),
			},
			wantError: false,
		},
		{
			name: "partial config with only confidence",
			config: &VADConfig{
				Confidence: float64Ptr(0.5),
			},
			wantError: false,
		},
		{
			name: "large valid start_secs",
			config: &VADConfig{
				StartSecs: float64Ptr(10.0),
			},
			wantError: false,
		},
		{
			name: "large valid stop_secs",
			config: &VADConfig{
				StopSecs: float64Ptr(5.0),
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if tt.wantError && tt.errorMsg != "" && err != nil {
				if err.Error() != tt.errorMsg {
					t.Errorf("Validate() error message = %q, want %q", err.Error(), tt.errorMsg)
				}
			}
		})
	}
}
