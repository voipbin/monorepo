package request

import "github.com/gofrs/uuid"

// V1DataExtensionsPost is
// v1 data type request struct for
// /v1/extensions POST
type V1DataExtensionsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`

	DomainID uuid.UUID `json:"domain_id"` // will ge removed

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataExtensionsIDPut is
// v1 data type request struct for
// /v1/extensions/{id} PUT
//
// Every field is a pointer: nil means "leave the existing value
// untouched", matching the OpenAPI-layer contract. See
// docs/plans/2026-09-12-registrar-put-partial-update-phase1-design.md.
type V1DataExtensionsIDPut struct {
	Name     *string `json:"name,omitempty"`
	Detail   *string `json:"detail,omitempty"`
	Password *string `json:"password,omitempty"`
}

// V1DataExtensionsExtensionExtensionGet is
// v1 data type request struct for
// /v1/extensions/extension/{extension} GET
type V1DataExtensionsExtensionExtensionGet struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
}
