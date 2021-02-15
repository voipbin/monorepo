package request

import "github.com/gofrs/uuid"

// RMV1DataDomainsPost is
// v1 data type request struct for
// /v1/domains POST to registrar-manager
type RMV1DataDomainsPost struct {
	UserID     uint64 `json:"user_id"`
	DomainName string `json:"domain_name"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// RMV1DataDomainsIDPut is
// v1 data type request struct for
// /v1/domains/{id} PUT
type RMV1DataDomainsIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
}

// RMV1DataExtensionsPost is
// v1 data type request struct for
// /v1/extensions POST to registrar-manager
type RMV1DataExtensionsPost struct {
	UserID uint64 `json:"user_id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`
}

// RMV1DataExtensionsIDPut is
// v1 data type request struct for
// /v1/extensions/{id} PUT
type RMV1DataExtensionsIDPut struct {
	Name     string `json:"name"`   // name
	Detail   string `json:"detail"` // detail
	Password string `json:"password"`
}
