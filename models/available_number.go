package models

// AvailableNumber struct represent number information
type AvailableNumber struct {
	Number string `json:"number"`

	ProviderName NumberProviderName `json:"provider_name"`

	Country    string                   `json:"country"`
	Region     string                   `json:"region"`
	PostalCode string                   `json:"postal_code"`
	Features   []AvailableNumberFeature `json:"features"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// AvailableNumberFeature type
type AvailableNumberFeature string

// list of AvailableNumberFeature
const (
	AvailableNumberFeatureEmergency AvailableNumberFeature = "emergency"
	AvailableNumberFeatureFax       AvailableNumberFeature = "fax"
	AvailableNumberFeatureMMS       AvailableNumberFeature = "mms"
	AvailableNumberFeatureSMS       AvailableNumberFeature = "sms"
	AvailableNumberFeatureVoice     AvailableNumberFeature = "voice"
)
