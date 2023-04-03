package request

// BodyDomainsPOST is rquest body define for
// POST /v1.0/domains
type BodyDomainsPOST struct {
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	DomainName string `json:"domain_name"`
}

// ParamDomainsGET is rquest param define for
// GET /v1.0/domains
type ParamDomainsGET struct {
	Pagination
}

// BodyDomainsIDPUT is rquest body define for
// PUT /v1.0/domains/<domain-id>
type BodyDomainsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}
