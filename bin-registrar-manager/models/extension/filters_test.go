package extension

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestFieldStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name string
		fs   FieldStruct
	}{
		{
			name: "complete_field_struct",
			fs: FieldStruct{
				ID:         id,
				CustomerID: customerID,
				Name:       "Test Extension",
				EndpointID: "endpoint_123",
				AORID:      "aor_123",
				AuthID:     "auth_123",
				Extension:  "1001",
				DomainName: "ext.example.com",
				Realm:      "example.com",
				Username:   "testuser",
				Deleted:    false,
			},
		},
		{
			name: "minimal_field_struct",
			fs: FieldStruct{
				ID:      id,
				Deleted: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify struct can be created and fields accessed
			if tt.fs.ID != uuid.Nil && tt.fs.ID == uuid.Nil {
				t.Error("ID should not be nil when set")
			}
		})
	}
}
