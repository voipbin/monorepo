package identity

import "github.com/gofrs/uuid"

// Identity represents
type Identity struct {
	// identity
	ID         uuid.UUID `json:"id" db:"id,uuid"`                    // resource identifier
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"` // resource's customer id
}
