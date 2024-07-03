package identity

import "github.com/gofrs/uuid"

// Identity represents
type Identity struct {
	// identity
	ID         uuid.UUID `json:"id"`          // resource identifier
	CustomerID uuid.UUID `json:"customer_id"` // resource's customer id
	OwnerType  OwnerType `json:"owner_type"`  // resource's owner type
	OwnerID    uuid.UUID `json:"owner_id"`    // resource's owner id
}

// OwnerType defines
type OwnerType string

// list of owner types
const (
	OwnerTypeNone  OwnerType = ""
	OwnerTypeAgent OwnerType = "agent" // the owner id is agent's id.
)
