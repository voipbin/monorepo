package request

import (
	"github.com/gofrs/uuid"
)

// V1DataContactsGet is
// v1 data type request struct for
// /v1/contacts GET
type V1DataContactsGet struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Extension  string    `json:"extension,omitempty"`
}

// V1DataContactsPut is
// v1 data type request struct for
// /v1/contacts PUT
type V1DataContactsPut struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Extension  string    `json:"extension,omitempty"`
}
