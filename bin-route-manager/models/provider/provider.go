package provider

import "github.com/gofrs/uuid"

// Provider defines
type Provider struct {
	ID uuid.UUID `json:"id" db:"id,uuid"`

	Type     Type   `json:"type" db:"type"`
	Hostname string `json:"hostname" db:"hostname"` // destination

	// sip type techs
	TechPrefix  string            `json:"tech_prefix" db:"tech_prefix"`   // tech prefix. valid only for the sip type.
	TechPostfix string            `json:"tech_postfix" db:"tech_postfix"` // tech postfix. valid only for the sip type.
	TechHeaders map[string]string `json:"tech_headers" db:"tech_headers,json"` // tech headers. valid only for the sip type.

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	// timestamp
	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// Type defines provider's type.
type Type string

// list of types
const (
	TypeSIP = "sip" // VoIP/SIP provider
)
