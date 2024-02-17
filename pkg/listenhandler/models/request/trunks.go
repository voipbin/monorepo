package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
)

// V1DataTrunksPost is
// v1 data type request struct for
// /v1/trunks POST
type V1DataTrunksPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	DomainName string             `json:"domain_name,omitempty"`
	Authtypes  []sipauth.AuthType `json:"auth_types,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	AllowedIPs []string `json:"allowed_ips,omitempty"`
}

// V1DataTrunksIDPut is
// v1 data type request struct for
// /v1/trunks/{id} PUT
type V1DataTrunksIDPut struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Authtypes []sipauth.AuthType `json:"auth_types,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	AllowedIPs []string `json:"allowed_ips,omitempty"`
}
