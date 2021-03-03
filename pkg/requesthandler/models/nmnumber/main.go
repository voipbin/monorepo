package nmnumber

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// AvailableNumber struct represent number information
type AvailableNumber struct {
	Number string `json:"number"`

	ProviderName string `json:"provider_name"`

	Country    string   `json:"country"`
	Region     string   `json:"region"`
	PostalCode string   `json:"postal_code"`
	Features   []string `json:"features"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertNumber returns converted data from number.Available
func (t *AvailableNumber) ConvertNumber() *models.AvailableNumber {

	res := &models.AvailableNumber{
		Number:     t.Number,
		Country:    t.Country,
		Region:     t.Region,
		PostalCode: t.PostalCode,
		Features:   t.Features,
	}

	return res
}
