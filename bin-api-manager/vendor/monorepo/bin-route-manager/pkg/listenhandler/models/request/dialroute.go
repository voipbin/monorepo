package request

import (
	"github.com/gofrs/uuid"
)

// V1DataDialroutesGet is
// v1 data type request struct for
// /v1/dialroutes GET
type V1DataDialroutesGet struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Target     string    `json:"target,omitempty"`
}
