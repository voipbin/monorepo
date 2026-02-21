package sipauth

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for SipAuth queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	Realm         string        `filter:"realm"`
	Username      string        `filter:"username"`
}
