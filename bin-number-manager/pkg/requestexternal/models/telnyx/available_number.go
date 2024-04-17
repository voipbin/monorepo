package telnyx

import (
	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
)

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
func (t *AvailableNumber) ConvertAvailableNumber() *availablenumber.AvailableNumber {

	res := &availablenumber.AvailableNumber{
		Number:       t.PhoneNumber,
		ProviderName: number.ProviderNameTelnyx,
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
		case string(availablenumber.FeatureEmergency):
			res.Features = append(res.Features, availablenumber.FeatureEmergency)

		case string(availablenumber.FeatureFax):
			res.Features = append(res.Features, availablenumber.FeatureFax)

		case string(availablenumber.FeatureMMS):
			res.Features = append(res.Features, availablenumber.FeatureMMS)

		case string(availablenumber.FeatureSMS):
			res.Features = append(res.Features, availablenumber.FeatureSMS)

		case string(availablenumber.FeatureVoice):
			res.Features = append(res.Features, availablenumber.FeatureVoice)
		}
	}

	return res
}
