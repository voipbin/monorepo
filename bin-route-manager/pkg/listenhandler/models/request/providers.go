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
	Codecs string `json:"codecs"`
}

// V1DataProvidersIDPut is
// v1 data type request struct for
// /v1/providers/{id} PUT
// All fields are optional pointers: nil means "unchanged", a non-nil
// pointer (including a pointer to an empty string) means "set to this value".
type V1DataProvidersIDPut struct {
	Type *provider.Type `json:"type,omitempty"`

	Hostname *string `json:"hostname,omitempty"`

	TechPrefix  *string            `json:"tech_prefix,omitempty"`
	TechPostfix *string            `json:"tech_postfix,omitempty"`
	TechHeaders *map[string]string `json:"tech_headers,omitempty"`

	Name   *string `json:"name,omitempty"`
	Detail *string `json:"detail,omitempty"`
	Codecs *string `json:"codecs,omitempty"`
}
