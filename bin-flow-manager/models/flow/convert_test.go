package flow

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		wantErr bool
	}{
		{
			name: "valid_fields",
			input: map[string]any{
				"id":          uuid.Must(uuid.NewV4()).String(),
				"customer_id": uuid.Must(uuid.NewV4()).String(),
				"type":        "flow",
				"name":        "test-flow",
				"detail":      "test detail",
			},
			wantErr: false,
		},
		{
			name: "empty_map",
			input: map[string]any{},
			wantErr: false,
		},
		{
			name: "partial_fields",
			input: map[string]any{
				"name":   "test-flow",
				"detail": "test detail",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertStringMapToFieldMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("ConvertStringMapToFieldMap() returned nil result when expecting success")
			}
		})
	}
}
