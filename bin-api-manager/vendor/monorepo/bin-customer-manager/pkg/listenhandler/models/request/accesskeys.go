package request

import (
	"github.com/gofrs/uuid"
)

// V1DataAccesskeysPost is
// v1 data type request struct for
// /v1/accesskeys POST
type V1DataAccesskeysPost struct {
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Expire int32 `json:"expire,omitempty"` // expiration in seconds
}

// V1DataAccesskeysIDPut is
// v1 data type request struct for
// /v1/accesskeys/<accesskey-id> PUT
type V1DataAccesskeysIDPut struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`
}
