package extension

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

// Extension struct
type Extension struct {
	commonidentity.Identity

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	// asterisk resources
	EndpointID string `json:"endpoint_id" db:"endpoint_id"`
	AORID      string `json:"aor_id" db:"aor_id"`
	AuthID     string `json:"auth_id" db:"auth_id"`

	Extension string `json:"extension" db:"extension"`

	DomainName string `json:"domain_name" db:"domain_name"`

	Realm    string `json:"realm" db:"realm"`       // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username string `json:"username" db:"username"` // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password string `json:"password" db:"password"` // DO NOT CHANGE. This used by the kamailio's INVITE validation

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
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
