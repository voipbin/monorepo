package request

import (
	"monorepo/bin-route-manager/models/provider"
)

// V1DataProvidersPost is
// v1 data type request struct for
// /v1/providers POST
type V1DataProvidersPost struct {
	Type provider.Type `json:"type"`

	Hostname string `json:"hostname"`

	TechPrefix  string            `json:"tech_prefix"`
	TechPostfix string            `json:"tech_postfix"`
	TechHeaders map[string]string `json:"tech_headers"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataProvidersIDPut is
// v1 data type request struct for
// /v1/providers/{id} PUT
type V1DataProvidersIDPut struct {
	Type provider.Type `json:"type"`

	Hostname string `json:"hostname"`

	TechPrefix  string            `json:"tech_prefix"`
	TechPostfix string            `json:"tech_postfix"`
	TechHeaders map[string]string `json:"tech_headers"`

	Name   string `json:"name"`
	Detail string `json:"detail"`
}
