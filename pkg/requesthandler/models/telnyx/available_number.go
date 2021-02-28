package telnyx

import "gitlab.com/voipbin/bin-manager/number-manager.git/models"

// AvailableNumber type
type AvailableNumber struct {
	PhoneNumber       string                       `json:"phone_number"`
	Reservable        bool                         `json:"reservable"`
	QuickShip         bool                         `json:"quickship"`
	VanityFormat      string                       `json:"vanity_format"`
	RecordType        string                       `json:"record_type"`
	CostInformation   AvailableCostInformation     `json:"cost_information"`
	BestEffort        bool                         `json:"best_effort"`
	Features          []AvailableFeature           `json:"features"`
	RegionInformation []AvailableRegionInformation `json:"region_information"`
}

// AvailableCostInformation struct
type AvailableCostInformation struct {
	MonthlyCost string `json:"monthly_cost"`
	UpfrontCost string `json:"upfront_cost"`
	Currency    string `json:"currency"`
}

// AvailableFeature struct
type AvailableFeature struct {
	Name string `json:"name"`
}

// AvailableRegionInformation struct
type AvailableRegionInformation struct {
	RegionName string `json:"region_name"`
	RegionType string `json:"region_type"`
}

// AvailableMetaData struct
type AvailableMetaData struct {
	TotalResults      int `json:"total_results"`
	BestEffortResults int `json:"best_effort_results"`
}

// ConvertAvailableNumber returns converted number
func (t *AvailableNumber) ConvertAvailableNumber() *models.AvailableNumber {

	res := &models.AvailableNumber{
		Number:       t.PhoneNumber,
		ProviderName: models.NumberProviderNameTelnyx,
	}

	for _, tmp := range t.RegionInformation {
		if tmp.RegionType == "country_code" {
			res.Country = tmp.RegionName
		} else if tmp.RegionType == "state" {
			res.Region = tmp.RegionName
		}
	}

	for _, tmp := range t.Features {
		switch tmp.Name {
		case string(models.AvailableNumberFeatureEmergency):
			res.Features = append(res.Features, models.AvailableNumberFeatureEmergency)

		case string(models.AvailableNumberFeatureFax):
			res.Features = append(res.Features, models.AvailableNumberFeatureFax)

		case string(models.AvailableNumberFeatureMMS):
			res.Features = append(res.Features, models.AvailableNumberFeatureMMS)

		case string(models.AvailableNumberFeatureSMS):
			res.Features = append(res.Features, models.AvailableNumberFeatureSMS)

		case string(models.AvailableNumberFeatureVoice):
			res.Features = append(res.Features, models.AvailableNumberFeatureVoice)
		}
	}

	return res
}
