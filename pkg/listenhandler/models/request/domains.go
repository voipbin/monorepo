package request

import "github.com/gofrs/uuid"

// V1DataDomainsPost is
// v1 data type request struct for
// /v1/domains POST
type V1DataDomainsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	DomainName string    `json:"domain_name"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataDomainsIDPut is
// v1 data type request struct for
// /v1/domains/{id} PUT
type V1DataDomainsIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
