package identity

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestIdentityStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	i := Identity{
		ID:         id,
		CustomerID: customerID,
	}

	if i.ID != id {
		t.Errorf("Identity.ID = %v, expected %v", i.ID, id)
	}
	if i.CustomerID != customerID {
		t.Errorf("Identity.CustomerID = %v, expected %v", i.CustomerID, customerID)
	}
}

func TestIdentityWithNilUUID(t *testing.T) {
	i := Identity{}

	if i.ID != uuid.Nil {
		t.Errorf("Identity.ID = %v, expected %v", i.ID, uuid.Nil)
	}
	if i.CustomerID != uuid.Nil {
		t.Errorf("Identity.CustomerID = %v, expected %v", i.CustomerID, uuid.Nil)
	}
}

func TestMultipleIdentities(t *testing.T) {
	tests := []struct {
		name       string
		id         uuid.UUID
		customerID uuid.UUID
	}{
		{"first_identity", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		{"second_identity", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
		{"third_identity", uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Identity{
				ID:         tt.id,
				CustomerID: tt.customerID,
			}
			if i.ID != tt.id {
				t.Errorf("Identity.ID = %v, expected %v", i.ID, tt.id)
			}
			if i.CustomerID != tt.customerID {
				t.Errorf("Identity.CustomerID = %v, expected %v", i.CustomerID, tt.customerID)
			}
		})
	}
}
