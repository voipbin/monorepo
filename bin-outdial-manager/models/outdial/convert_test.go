package outdial

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConvertStringMapToFieldMap(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]any
		expectErr bool
	}{
		{
			name: "converts valid string map with UUID fields",
			input: map[string]any{
				"id":          uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000").String(),
				"customer_id": uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001").String(),
				"campaign_id": uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002").String(),
				"name":        "Test Outdial",
				"detail":      "Test Detail",
			},
			expectErr: false,
		},
		{
			name: "converts empty map",
			input: map[string]any{},
			expectErr: false,
		},
		{
			name: "converts map with string fields only",
			input: map[string]any{
				"name":   "Test",
				"detail": "Detail",
				"data":   "Some data",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertStringMapToFieldMap(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result == nil {
					t.Error("Expected result but got nil")
				}
				if len(result) != len(tt.input) {
					t.Errorf("Expected %d fields, got %d", len(tt.input), len(result))
				}
			}
		})
	}
}

func TestConvertStringMapToFieldMap_UUIDConversion(t *testing.T) {
	testUUID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000")

	input := map[string]any{
		"id": testUUID.String(),
	}

	result, err := ConvertStringMapToFieldMap(input)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 field, got %d", len(result))
	}

	// Verify the field exists
	if _, ok := result[Field("id")]; !ok {
		t.Error("Expected 'id' field in result")
	}
}
