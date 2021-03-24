package availablenumber

import nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

// AvailableNumber struct represent number information
type AvailableNumber struct {
	Number string `json:"number"`

	Country    string   `json:"country"`
	Region     string   `json:"region"`
	PostalCode string   `json:"postal_code"`
	Features   []string `json:"features"`
}

// ConvertNumber returns converted data from number.Available
func ConvertNumber(t *nmavailablenumber.AvailableNumber) *AvailableNumber {

	fetures := []string{}
	for _, feature := range t.Features {
		fetures = append(fetures, string(feature))
	}

	res := &AvailableNumber{
		Number:     t.Number,
		Country:    t.Country,
		Region:     t.Region,
		PostalCode: t.PostalCode,
		Features:   fetures,
	}

	return res
}
