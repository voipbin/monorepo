package queue

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	customerID := uuid.Must(uuid.NewV4())
	queueID := uuid.Must(uuid.NewV4())
	waitFlowID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name      string
		input     map[string]any
		expectErr bool
		validate  func(t *testing.T, result map[Field]any)
	}{
		{
			name: "valid_conversion_with_all_fields",
			input: map[string]any{
				"id":                      queueID.String(),
				"customer_id":             customerID.String(),
				"name":                    "Test Queue",
				"detail":                  "Test Detail",
				"routing_method":          "random",
				"execute":                 "run",
				"wait_flow_id":            waitFlowID.String(),
				"wait_timeout":            60000,
				"service_timeout":         300000,
				"total_incoming_count":    100,
				"total_serviced_count":    80,
				"total_abandoned_count":   20,
			},
			expectErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if result[FieldName] != "Test Queue" {
					t.Errorf("Name mismatch: got %v", result[FieldName])
				}
				if result[FieldDetail] != "Test Detail" {
					t.Errorf("Detail mismatch: got %v", result[FieldDetail])
				}
				if result[FieldWaitTimeout] != 60000 {
					t.Errorf("WaitTimeout mismatch: got %v", result[FieldWaitTimeout])
				}
			},
		},
		{
			name: "valid_conversion_partial_fields",
			input: map[string]any{
				"name":   "Partial Queue",
				"detail": "Partial Detail",
			},
			expectErr: false,
			validate: func(t *testing.T, result map[Field]any) {
				if result[FieldName] != "Partial Queue" {
					t.Errorf("Name mismatch: got %v", result[FieldName])
				}
				if result[FieldDetail] != "Partial Detail" {
					t.Errorf("Detail mismatch: got %v", result[FieldDetail])
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
				"wait_timeout": "not_a_number",
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
