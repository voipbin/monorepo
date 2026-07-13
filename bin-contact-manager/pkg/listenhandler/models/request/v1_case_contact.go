package request

import "github.com/gofrs/uuid"

// V1DataCasesIDPut is the request body for PUT /v1/cases/{id}.
type V1DataCasesIDPut struct {
	CustomerID uuid.UUID `json:"customer_id"`
	ContactID  uuid.UUID `json:"contact_id"` // uuid.Nil clears the attribution
}
