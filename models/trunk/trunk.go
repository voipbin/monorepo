package trunk

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/sipauth"
)

// Trunk struct
type Trunk struct {
	ID         uuid.UUID `json:"id,omitempty"`
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	DomainName string `json:"domain_name,omitempty"`

	// sip info
	AuthTypes  []sipauth.AuthType `json:"auth_types,omitempty"`  // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Realm      string             `json:"realm,omitempty"`       // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username   string             `json:"username,omitempty"`    // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password   string             `json:"password,omitempty"`    // DO NOT CHANGE. This used by the kamailio's INVITE validation
	AllowedIPs []string           `json:"allowed_ips,omitempty"` // DO NOT CHANGE. This used by the kamailio's INVITE validation

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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
