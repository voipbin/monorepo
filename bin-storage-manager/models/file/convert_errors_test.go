package file

import (
	"strings"
	"testing"
)

// Test_ConvertStringMapToFieldMap_invalidValue checks that a value which cannot be converted
// to its field type is rejected, and that the error names the field.
func Test_ConvertStringMapToFieldMap_invalidValue(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]any
		expect string
	}{
		{"uuid field with a malformed id", map[string]any{"customer_id": "not-a-uuid"}, "field customer_id"},
		{"int field with a non-numeric string", map[string]any{"filesize": "abc"}, "field filesize"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ConvertStringMapToFieldMap(tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.expect) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expect, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil map, got: %v", res)
			}
		})
	}
}
