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
type V1DataExtensionsIDPut struct {
	Name             string `json:"name"`
	Detail           string `json:"detail"`
	Password         string `json:"password"`
	Direct           *bool  `json:"direct,omitempty"`
	DirectRegenerate *bool  `json:"direct_regenerate,omitempty"`
}

// V1DataExtensionsExtensionExtensionGet is
// v1 data type request struct for
// /v1/extensions/extension/{extension} GET
type V1DataExtensionsExtensionExtensionGet struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
}
