package trunk

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
)

// Trunk struct
type Trunk struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	DomainName string `json:"domain_name"`

	// sip info
	AuthTypes  []sipauth.AuthType `json:"auth_types"`  // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Realm      string             `json:"realm"`       // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username   string             `json:"username"`    // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password   string             `json:"password"`    // DO NOT CHANGE. This used by the kamailio's INVITE validation
	AllowedIPs []string           `json:"allowed_ips"` // DO NOT CHANGE. This used by the kamailio's INVITE validation

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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
