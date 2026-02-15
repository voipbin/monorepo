package queuecall

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	queueID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		input     map[string]any
		expectErr bool
		validate  func(t *testing.T, result map[Field]any)
	}{
		{
			name: "valid_conversion_with_all_fields",
			input: map[string]any{
				"customer_id":      customerID.String(),
				"queue_id":         queueID.String(),
				"reference_type":   "call",
				"reference_id":     referenceID.String(),
				"status":           "waiting",
				"timeout_wait":     30000,
				"timeout_service":  60000,
				"duration_waiting": 5000,
				"duration_service": 10000,
			},
			expectErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if result[FieldStatus] != "waiting" {
					t.Errorf("Status mismatch: got %v", result[FieldStatus])
				}
				if result[FieldTimeoutWait] != 30000 {
					t.Errorf("TimeoutWait mismatch: got %v", result[FieldTimeoutWait])
				}
			},
		},
		{
			name: "valid_conversion_partial_fields",
			input: map[string]any{
				"status": "serviced",
			},
			expectErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if result[FieldStatus] != "serviced" {
					t.Errorf("Status mismatch: got %v", result[FieldStatus])
				}
			},
		},
		{
			name: "empty_map",
			input: map[string]any{},
			expectErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if len(result) != 0 {
					t.Errorf("Expected empty result, got %d fields", len(result))
				}
			},
		},
		{
			name: "invalid_field_type",
			input: map[string]any{
				"timeout_wait": "not_a_number",
			},
			expectErr: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
