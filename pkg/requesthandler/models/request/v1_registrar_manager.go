package request

// RMV1DataDomainsPost is
// v1 data type request struct for
// /v1/domains POST to registrar-manager
type RMV1DataDomainsPost struct {
	UserID     uint64 `json:"user_id"`
	DomainName string `json:"domain_name"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// RMV1DataDomainIDPut is
// v1 data type request struct for
// /v1/domains/{id} PUT
type RMV1DataDomainIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
}
