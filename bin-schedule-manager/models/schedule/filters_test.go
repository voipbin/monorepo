package schedule

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name string

		field FieldStruct

		expectCID     uuid.UUID
		expectEnabled bool
		expectDeleted bool
	}{
		{
			name: "field_struct_with_all_fields",

			field: FieldStruct{
				CustomerID: uuid.FromStringOrNil("0693a45c-6f14-11f0-8683-1b39a30d5e52"),
				Enabled:    true,
				Deleted:    false,
			},

			expectCID:     uuid.FromStringOrNil("0693a45c-6f14-11f0-8683-1b39a30d5e52"),
			expectEnabled: true,
			expectDeleted: false,
		},
		{
			name: "field_struct_with_deleted_true",

			field: FieldStruct{
				CustomerID: uuid.FromStringOrNil("06c37b0a-6f14-11f0-b7dc-8fb56f57b0ce"),
				Enabled:    false,
				Deleted:    true,
			},

			expectCID:     uuid.FromStringOrNil("06c37b0a-6f14-11f0-b7dc-8fb56f57b0ce"),
			expectEnabled: false,
			expectDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.CustomerID != tt.expectCID {
				t.Errorf("Wrong CustomerID. expect: %s, got: %s", tt.expectCID, tt.field.CustomerID)
			}
			if tt.field.Enabled != tt.expectEnabled {
				t.Errorf("Wrong Enabled. expect: %v, got: %v", tt.expectEnabled, tt.field.Enabled)
			}
			if tt.field.Deleted != tt.expectDeleted {
				t.Errorf("Wrong Deleted. expect: %v, got: %v", tt.expectDeleted, tt.field.Deleted)
			}
		})
	}
}
