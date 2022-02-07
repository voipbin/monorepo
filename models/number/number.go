package number

import "github.com/gofrs/uuid"

// Number struct represent number information
type Number struct {
	ID         uuid.UUID `json:"id"`
	Number     string    `json:"number"`
	FlowID     uuid.UUID `json:"flow_id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ProviderName        ProviderName `json:"provider_name"`
	ProviderReferenceID string       `json:"provider_reference_id"`

	Status Status `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	// timestamp
	TMPurchase string `json:"tm_purchase"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ProviderName type
type ProviderName string

// list of NumberProvider
const (
	ProviderNameTelnyx      ProviderName = "telnyx"
	ProviderNameTwilio      ProviderName = "twilio"
	ProviderNameMessagebird ProviderName = "messagebird"
)

// Status type
type Status string

// List of NumberStatus types
const (
	StatusActive  Status = "active"
	StatusDeleted Status = "deleted"
)
