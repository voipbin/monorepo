package outplan

import (
	"testing"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name    string
		src     map[string]any
		wantErr bool
	}{
		{
			name: "valid_conversion",
			src: map[string]any{
				"name":           "Test Outplan",
				"detail":         "Test detail",
				"dial_timeout":   30000,
				"max_try_count_0": 3,
			},
			wantErr: false,
		},
		{
			name: "valid_conversion_with_all_fields",
			src: map[string]any{
				"name":            "Full Outplan",
				"detail":          "Complete test",
				"dial_timeout":    60000,
				"try_interval":    5000,
				"max_try_count_0": 1,
				"max_try_count_1": 2,
				"max_try_count_2": 3,
				"max_try_count_3": 4,
				"max_try_count_4": 5,
			},
			wantErr: false,
		},
		{
			name:    "empty_map",
			src:     map[string]any{},
			wantErr: false,
		},
		{
			name: "valid_conversion_minimal",
			src: map[string]any{
				"name": "Minimal Outplan",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertStringMapToFieldMap(tt.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringMapToFieldMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("ConvertStringMapToFieldMap() returned nil result without error")
			}
			if !tt.wantErr && len(got) != len(tt.src) {
				t.Errorf("ConvertStringMapToFieldMap() result length = %v, expected %v", len(got), len(tt.src))
			}
		})
	}
}

func TestMaxTryCountLen(t *testing.T) {
	if MaxTryCountLen != 5 {
		t.Errorf("MaxTryCountLen = %v, expected 5", MaxTryCountLen)
	}
}
