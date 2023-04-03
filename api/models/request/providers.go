package request

import (
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
)

// ParamProvidersGET is request param define for
// GET /v1.0/providers
type ParamProvidersGET struct {
	Pagination
}

// BodyProvidersPOST is request body define for
// POST /v1.0/providers
type BodyProvidersPOST struct {
	Type        rmprovider.Type   `json:"type"`
	Hostname    string            `json:"hostname"`
	TechPrefix  string            `json:"tech_prefix"`
	TechPostfix string            `json:"tech_postfix"`
	TechHeaders map[string]string `json:"tech_headers"`
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
}

// BodyProvidersIDPUT is request body define for
// PUT /v1.0/providers/<provider-id>
type BodyProvidersIDPUT struct {
	Type        rmprovider.Type   `json:"type"`
	Hostname    string            `json:"hostname"`
	TechPrefix  string            `json:"tech_prefix"`
	TechPostfix string            `json:"tech_postfix"`
	TechHeaders map[string]string `json:"tech_headers"`
	Name        string            `json:"name"`
	Detail      string            `json:"detail"`
}
