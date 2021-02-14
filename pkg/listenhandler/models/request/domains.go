package request

// V1DataDomainsPost is
// v1 data type request struct for
// /v1/domains POST
type V1DataDomainsPost struct {
	UserID     uint64 `json:"user_id"`
	DomainName string `json:"domain_name"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataDomainsPut is
// v1 data type request struct for
// /v1/domains PUT
type V1DataDomainsPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
