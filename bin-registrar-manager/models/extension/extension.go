package extension

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

// Extension struct
type Extension struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Detail string `json:"detail"`

	// asterisk resources
	EndpointID string `json:"endpoint_id"`
	AORID      string `json:"aor_id"`
	AuthID     string `json:"auth_id"`

	Extension string `json:"extension"`

	DomainName string `json:"domain_name"`

	Realm    string `json:"realm"`    // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username string `json:"username"` // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password string `json:"password"` // DO NOT CHANGE. This used by the kamailio's INVITE validation

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// GenerateSIPAuth returns sipauth of the given extension
func (h *Extension) GenerateSIPAuth() *sipauth.SIPAuth {
	return &sipauth.SIPAuth{
		ID:            h.ID,
		ReferenceType: sipauth.ReferenceTypeExtension,

		AuthTypes:  []sipauth.AuthType{sipauth.AuthTypeBasic},
		Realm:      h.Realm,
		Username:   h.Username,
		Password:   h.Password,
		AllowedIPs: []string{},
	}
}
