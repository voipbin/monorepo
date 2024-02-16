package trunk

import "github.com/gofrs/uuid"

// Trunk struct
type Trunk struct {
	ID         uuid.UUID `json:"id,omitempty"`
	CustomerID uuid.UUID `json:"customer_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	DomainName string `json:"domain_name,omitempty"`

	AuthTypes []AuthType `json:"auth_types,omitempty"` // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Realm     string     `json:"realm,omitempty"`      // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Username  string     `json:"username,omitempty"`   // DO NOT CHANGE. This used by the kamailio's INVITE validation
	Password  string     `json:"password,omitempty"`   // DO NOT CHANGE. This used by the kamailio's INVITE validation

	AllowedIPs []string `json:"allowed_ips,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// AuthType define
type AuthType string

// list of AuthType types
const (
	AuthTypeBasic AuthType = "basic" // basic authentication
	AuthTypeIP    AuthType = "ip"    // ip
)
