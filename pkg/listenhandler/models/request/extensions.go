package request

import "github.com/gofrs/uuid"

// V1DataExtensionsPost is
// v1 data type request struct for
// /v1/extensions POST
type V1DataExtensionsPost struct {
	UserID uint64 `json:"user_id"`

	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}
