package provider

import "github.com/gofrs/uuid"

// Provider defines
type Provider struct {
	ID uuid.UUID `json:"id"`

	Type     Type   `json:"type"`
	Hostname string `json:"hostname"` // destination

	// sip type techs
	TechPrefix  string            `json:"tech_prefix"`  // tech prefix. valid only for the sip type.
	TechPostfix string            `json:"tech_postfix"` // tech postfix. valid only for the sip type.
	TechHeaders map[string]string `json:"tech_headers"` // tech headers. valid only for the sip type.

	Name   string `json:"name"`
	Detail string `json:"detail"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type defines provider's type.
type Type string

// list of types
const (
	TypeSIP = "sip" // VoIP/SIP provider
)
