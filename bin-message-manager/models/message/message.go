package message

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-message-manager/models/target"
)

// Message defines
type Message struct {
	commonidentity.Identity

	Type Type `json:"type"`

	// from/to info
	Source  *commonaddress.Address `json:"source"`
	Targets []target.Target        `json:"targets"`

	// provider info
	ProviderName        ProviderName `json:"provider_name"`
	ProviderReferenceID string       `json:"provider_reference_id"`

	// message info
	Text      string    `json:"text"` // Text delivered in the body of the message.
	Medias    []string  `json:"medias"`
	Direction Direction `json:"direction"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
	TypeSMS Type = "sms"
)

// Direction defines
type Direction string

// list of Directions types
const (
	DirectionOutbound Direction = "outbound" // direction outbound.
	DirectionInbound  Direction = "inbound"  // direction inbound
)

// ProviderName type
type ProviderName string

// list of NumberProvider
const (
	ProviderNameTelnyx      ProviderName = "telnyx"
	ProviderNameTwilio      ProviderName = "twilio"
	ProviderNameMessagebird ProviderName = "messagebird"
)
