package campaigncall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name    string
		src     map[string]any
		wantErr bool
	}{
		{
			name: "valid_conversion_with_uuid",
			src: map[string]any{
				"campaign_id": uuid.Must(uuid.NewV4()).String(),
				"status":      "dialing",
				"result":      "success",
			},
			wantErr: false,
		},
		{
			name: "valid_conversion_without_uuid",
			src: map[string]any{
				"status": "progressing",
				"result": "fail",
			},
			wantErr: false,
		},
		{
			name:    "empty_map",
			src:     map[string]any{},
			wantErr: false,
		},
		{
			name: "invalid_uuid_format",
			src: map[string]any{
				"campaign_id": "invalid-uuid-format",
			},
			wantErr: true,
		},
		{
			name: "multiple_uuids",
			src: map[string]any{
				"campaign_id":       uuid.Must(uuid.NewV4()).String(),
				"outplan_id":        uuid.Must(uuid.NewV4()).String(),
				"outdial_id":        uuid.Must(uuid.NewV4()).String(),
				"outdial_target_id": uuid.Must(uuid.NewV4()).String(),
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
