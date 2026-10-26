package twilio

import (
	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
)

// AvailableNumber is an item of the AvailablePhoneNumbers Local resource.
type AvailableNumber struct {
	FriendlyName string       `json:"friendly_name"`
	PhoneNumber  string       `json:"phone_number"`
	Locality     string       `json:"locality"`
	Region       string       `json:"region"`
	PostalCode   string       `json:"postal_code"`
	IsoCountry   string       `json:"iso_country"`
	Capabilities Capabilities `json:"capabilities"`
}

// Capabilities of a number
type Capabilities struct {
	Voice bool `json:"voice"`
	SMS   bool `json:"sms"`
	MMS   bool `json:"mms"`
	Fax   bool `json:"fax"`
}

// ResponseAvailableNumbers is the response of the AvailablePhoneNumbers Local read.
type ResponseAvailableNumbers struct {
	AvailablePhoneNumbers []AvailableNumber `json:"available_phone_numbers"`
}

// ConvertAvailableNumber returns the converted number
func (t *AvailableNumber) ConvertAvailableNumber() *availablenumber.AvailableNumber {
	res := &availablenumber.AvailableNumber{
		Number:       t.PhoneNumber,
		ProviderName: number.ProviderNameTwilio,
		Country:      t.IsoCountry,
		Region:       t.Region,
		PostalCode:   t.PostalCode,
	}

	if t.Capabilities.Voice {
		res.Features = append(res.Features, availablenumber.FeatureVoice)
	}
	if t.Capabilities.SMS {
		res.Features = append(res.Features, availablenumber.FeatureSMS)
	}
	if t.Capabilities.MMS {
		res.Features = append(res.Features, availablenumber.FeatureMMS)
	}
	if t.Capabilities.Fax {
		res.Features = append(res.Features, availablenumber.FeatureFax)
	}

	return res
}
