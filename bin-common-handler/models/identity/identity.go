package identity

import "github.com/gofrs/uuid"

// Identity represents
type Identity struct {
	// identity
	ID         uuid.UUID `json:"id"`          // resource identifier
	CustomerID uuid.UUID `json:"customer_id"` // resource's customer id
}
