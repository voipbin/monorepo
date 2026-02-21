package trunk

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-registrar-manager/models/sipauth"
)

// Trunk struct
type Trunk struct {
	commonidentity.Identity

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	DomainName string `json:"domain_name" db:"domain_name"`

	// sip info
	AuthTypes  []sipauth.AuthType `json:"auth_types" db:"auth_types,json"`  // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Realm      string             `json:"realm" db:"realm"`                 // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username   string             `json:"username" db:"username"`           // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password   string             `json:"password" db:"password"`           // DO NOT CHANGE. This used by the kamailio's INVITE validation
	AllowedIPs []string           `json:"allowed_ips" db:"allowed_ips,json"` // DO NOT CHANGE. This used by the kamailio's INVITE validation

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// GenerateSIPAuth returns sipauth of the given trunk
func (h *Trunk) GenerateSIPAuth() *sipauth.SIPAuth {
	return &sipauth.SIPAuth{
		ID:            h.ID,
		ReferenceType: sipauth.ReferenceTypeTrunk,

		AuthTypes:  h.AuthTypes,
		Realm:      h.Realm,
		Username:   h.Username,
		Password:   h.Password,
		AllowedIPs: h.AllowedIPs,
	}
}
