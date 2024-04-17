package request

import (
	rmsipauth "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
)

// BodyTrunksPOST is rquest body define for
// POST /v1.0/trunks
type BodyTrunksPOST struct {
	Name       string               `json:"name"`
	Detail     string               `json:"detail"`
	DomainName string               `json:"domain_name"`
	AuthTypes  []rmsipauth.AuthType `json:"auth_types"`
	Username   string               `json:"username"`
	Password   string               `json:"password"`
	AllowedIPs []string             `json:"allowed_ips"`
}

// ParamTrunksGET is rquest param define for
// GET /v1.0/trunks
type ParamTrunksGET struct {
	Pagination
}

// BodyTrunksIDPUT is rquest body define for
// PUT /v1.0/trunks/<trunk-id>
type BodyTrunksIDPUT struct {
	Name       string               `json:"name"`
	Detail     string               `json:"detail"`
	AuthTypes  []rmsipauth.AuthType `json:"auth_types"`
	Username   string               `json:"username"`
	Password   string               `json:"password"`
	AllowedIPs []string             `json:"allowed_ips"`
}
