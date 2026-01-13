package number

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Number struct represent number information
type Number struct {
	commonidentity.Identity

	Number string `json:"number" db:"number"`

	CallFlowID    uuid.UUID `json:"call_flow_id" db:"call_flow_id,uuid"`
	MessageFlowID uuid.UUID `json:"message_flow_id" db:"message_flow_id,uuid"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	ProviderName        ProviderName `json:"provider_name" db:"provider_name"`
	ProviderReferenceID string       `json:"provider_reference_id" db:"provider_reference_id"`

	Status Status `json:"status" db:"status"`

	T38Enabled       bool `json:"t38_enabled" db:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled" db:"emergency_enabled"`

	// timestamp
	TMPurchase string `json:"tm_purchase" db:"tm_purchase"`
	TMRenew    string `json:"tm_renew" db:"tm_renew"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// ProviderName type
type ProviderName string

// list of NumberProvider
const (
	ProviderNameNone        ProviderName = ""
	ProviderNameTelnyx      ProviderName = "telnyx"
	ProviderNameTwilio      ProviderName = "twilio"
	ProviderNameMessagebird ProviderName = "messagebird"
)

// Status type
type Status string

// List of NumberStatus types
const (
	StatusNone    Status = ""
	StatusActive  Status = "active"
	StatusDeleted Status = "deleted"
)
