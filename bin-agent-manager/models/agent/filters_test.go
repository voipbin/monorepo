package agent

import (
	"testing"

	"github.com/gofrs/uuid"
)

func Test_FieldStruct(t *testing.T) {
	// Test that FieldStruct can be instantiated with various field types
	tests := []struct {
		name   string
		fields FieldStruct
	}{
		{
			name: "all fields populated",
			fields: FieldStruct{
				ID:         uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				CustomerID: uuid.FromStringOrNil("33f9ca84-7fde-11ec-a186-9f2e8c3a62aa"),
				Username:   "test@voipbin.net",
				Name:       "test name",
				RingMethod: RingMethodRingAll,
				Status:     StatusAvailable,
				Deleted:    false,
			},
		},
		{
			name: "minimal fields",
			fields: FieldStruct{
				ID:      uuid.FromStringOrNil("4f6a7348-4b42-11ec-80ba-13dbc38fe32c"),
				Deleted: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify that the struct can be created
			if tt.fields.ID == uuid.Nil && tt.name == "all fields populated" {
				t.Errorf("Expected non-nil ID")
			}
		})
	}
}
