package request

// BodyDomainsPOST is rquest body define for POST /domains
type BodyDomainsPOST struct {
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	DomainName string `json:"domain_name"`
}

// ParamDomainsGET is rquest param define for GET /domains
type ParamDomainsGET struct {
	Pagination
}

// BodyDomainsIDPUT is rquest body define for PUT /domains/{id}
type BodyDomainsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
