package availablenumber

import (
	"monorepo/bin-number-manager/models/number"
)

// AvailableNumber struct represent number information
type AvailableNumber struct {
	Number string `json:"number"`

	ProviderName number.ProviderName `json:"provider_name"`

	Country    string    `json:"country"`
	Region     string    `json:"region"`
	PostalCode string    `json:"postal_code"`
	Features   []Feature `json:"features"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Feature type
type Feature string

// list of Feature
const (
	FeatureEmergency Feature = "emergency"
	FeatureFax       Feature = "fax"
	FeatureMMS       Feature = "mms"
	FeatureSMS       Feature = "sms"
	FeatureVoice     Feature = "voice"
)
