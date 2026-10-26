package twilio

import (
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
)

// emergencyStatusActive is the emergency_status of a number that can place emergency calls.
const emergencyStatusActive = "Active"

// IncomingPhoneNumber is a number of the account (the response of the purchase).
type IncomingPhoneNumber struct {
	SID             string `json:"sid"`
	PhoneNumber     string `json:"phone_number"`
	EmergencyStatus string `json:"emergency_status"`
}

// ConvertProviderNumber returns the converted ProviderNumber. The twilio number is active as soon
// as the purchase answers, and it has no T.38 fax gateway setting.
func (t *IncomingPhoneNumber) ConvertProviderNumber() *providernumber.ProviderNumber {
	return &providernumber.ProviderNumber{
		ID:               t.SID,
		Status:           number.StatusActive,
		T38Enabled:       false,
		EmergencyEnabled: t.EmergencyStatus == emergencyStatusActive,
	}
}
