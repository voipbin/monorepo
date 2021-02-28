package models

import "github.com/gofrs/uuid"

// Number struct represent number information
type Number struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	FlowID uuid.UUID `json:"flow_id"`
	UserID uint64    `json:"user_id"`

	ProviderName        NumberProviderName `json:"provider_name"`
	ProviderReferenceID string             `json:"provider_reference_id"`

	Status NumberStatus `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	// timestamp
	TMPurchase string `json:"tm_purchase"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// NumberProviderName type
type NumberProviderName string

// list of NumberProvider
const (
	NumberProviderNameTelnyx      NumberProviderName = "telnyx"
	NumberProviderNameTwilio      NumberProviderName = "twilio"
	NumberProviderNameMessagebird NumberProviderName = "messagebird"
)

// NumberStatus type
type NumberStatus string

// List of NumberStatus types
const (
	NumberStatusActive  NumberStatus = "active"
	NumberStatusDeleted NumberStatus = "deleted"
)
