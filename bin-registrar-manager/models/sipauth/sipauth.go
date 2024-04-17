package sipauth

import "github.com/gofrs/uuid"

// SIPAuth struct
type SIPAuth struct {
	ID            uuid.UUID     `json:"id,omitempty"`   // reference id
	ReferenceType ReferenceType `json:"type,omitempty"` // reference type

	AuthTypes []AuthType `json:"auth_types,omitempty"`
	Realm     string     `json:"realm,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	AllowedIPs []string `json:"allowed_ips,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
}

// ReferenceType define
type ReferenceType string

// list of Type types
const (
	ReferenceTypeTrunk     ReferenceType = "trunk"     // trunk
	ReferenceTypeExtension ReferenceType = "extension" // extension
)

// AuthType define
type AuthType string

// list of AuthType types
const (
	AuthTypeBasic AuthType = "basic" // basic authentication
	AuthTypeIP    AuthType = "ip"    // ip
)
