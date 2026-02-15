package tag

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	tests := []struct {
		name     string
		field    FieldStruct
		expectID uuid.UUID
		expectCID uuid.UUID
		expectName string
		expectDeleted bool
	}{
		{
			name: "field_struct_with_all_fields",
			field: FieldStruct{
				ID:         uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				CustomerID: uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:       "test name",
				Deleted:    false,
			},
			expectID:   uuid.FromStringOrNil("250bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			expectCID:  uuid.FromStringOrNil("350bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			expectName: "test name",
			expectDeleted: false,
		},
		{
			name: "field_struct_with_deleted_true",
			field: FieldStruct{
				ID:         uuid.FromStringOrNil("450bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				CustomerID: uuid.FromStringOrNil("550bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
				Name:       "deleted tag",
				Deleted:    true,
			},
			expectID:   uuid.FromStringOrNil("450bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			expectCID:  uuid.FromStringOrNil("550bbfa4-50d7-11ec-a6b1-8f9671a9e70e"),
			expectName: "deleted tag",
			expectDeleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.ID != tt.expectID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectID, tt.field.ID)
			}
			if tt.field.CustomerID != tt.expectCID {
				t.Errorf("Wrong CustomerID. expect: %s, got: %s", tt.expectCID, tt.field.CustomerID)
			}
			if tt.field.Name != tt.expectName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.expectName, tt.field.Name)
			}
			if tt.field.Deleted != tt.expectDeleted {
				t.Errorf("Wrong Deleted. expect: %v, got: %v", tt.expectDeleted, tt.field.Deleted)
			}
		})
	}
}
