package sipauth

import "github.com/gofrs/uuid"

// SIPAuth struct
type SIPAuth struct {
	ID            uuid.UUID     `json:"id,omitempty" db:"id,uuid"`         // reference id
	ReferenceType ReferenceType `json:"type,omitempty" db:"reference_type"` // reference type

	AuthTypes []AuthType `json:"auth_types,omitempty" db:"auth_types,json"`
	Realm     string     `json:"realm,omitempty" db:"realm"`

	Username string `json:"username,omitempty" db:"username"`
	Password string `json:"password,omitempty" db:"password"`

	AllowedIPs []string `json:"allowed_ips,omitempty" db:"allowed_ips,json"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
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
