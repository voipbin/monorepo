package identity

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
)

// The pointer type carries the promotable default subscription address (VOIP-1419): every model
// embedding Identity by value satisfies the interface through promotion.
var _ eventtopic.SubscriptionIdentifier = (*Identity)(nil)

// TestIdentityEventSubscriptionID pins the promotable default: the address IS the own id
// (mutation-checked against CustomerID, the one plausible wrong-answer field on Identity), and a
// zero value degrades to the uuid.Nil string, which downstream normalization turns into the `-`
// placeholder.
func TestIdentityEventSubscriptionID(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())

	i := &Identity{ID: id, CustomerID: customerID}

	if res := i.EventSubscriptionID(); res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
	if res := i.EventSubscriptionID(); res == customerID.String() {
		t.Errorf("Wrong match. the address must not be the customer id. got: %s", res)
	}

	zero := &Identity{}
	if res := zero.EventSubscriptionID(); res != uuid.Nil.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", uuid.Nil.String(), res)
	}
}

// TestIdentityEventSubscriptionIDPromotes pins the promotion mechanism itself: a struct that
// embeds Identity by value satisfies eventtopic.SubscriptionIdentifier through its pointer type
// with no method of its own -- the mechanism every default published model relies on.
func TestIdentityEventSubscriptionIDPromotes(t *testing.T) {
	type embedding struct {
		Identity
	}

	id := uuid.Must(uuid.NewV4())
	e := &embedding{Identity: Identity{ID: id}}

	var s eventtopic.SubscriptionIdentifier = e
	if res := s.EventSubscriptionID(); res != id.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", id.String(), res)
	}
}

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
