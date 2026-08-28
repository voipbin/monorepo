package identity

import "github.com/gofrs/uuid"

// Identity represents
type Identity struct {
	// identity
	ID         uuid.UUID `json:"id" db:"id,uuid"`                    // resource identifier
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"` // resource's customer id
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event`: the resource's own id (VOIP-1419). Every model that embeds Identity by
// value satisfies eventtopic.SubscriptionIdentifier through method promotion, so "own id" is the
// automatic default address of every published resource. A type whose address is NOT its own id
// (a stream child addressed by its parent, a wrapper that needs a nil-embed guard, a payload
// with no address at all) declares its own EventSubscriptionID, which shadows this one at a
// shallower depth. An empty or uuid.Nil return degrades to the `-` placeholder downstream.
//
// Pointer receiver on purpose: event data reaches notifyhandler.PublishEvent as a pointer, and
// a value embed of Identity inside an addressable outer value promotes this method into the
// outer pointer's method set.
func (h *Identity) EventSubscriptionID() string {
	return h.ID.String()
}
