package twilio

import (
	"encoding/json"
	"reflect"
	"testing"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
)

func Test_ConvertAvailableNumber(t *testing.T) {
	raw := `{"available_phone_numbers":[{"friendly_name":"x","phone_number":"+15550000001","locality":"L","region":"CA","postal_code":"94000","iso_country":"US","capabilities":{"voice":true,"SMS":true,"MMS":false,"fax":true}}]}`

	parsed := ResponseAvailableNumbers{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil || len(parsed.AvailablePhoneNumbers) != 1 {
		t.Fatalf("Wrong parse. got: %+v, %v", parsed, err)
	}

	res := parsed.AvailablePhoneNumbers[0].ConvertAvailableNumber()
	expect := &availablenumber.AvailableNumber{
		Number:       "+15550000001",
		ProviderName: number.ProviderNameTwilio,
		Country:      "US",
		Region:       "CA",
		PostalCode:   "94000",
		Features:     []availablenumber.Feature{availablenumber.FeatureVoice, availablenumber.FeatureSMS, availablenumber.FeatureFax},
	}
	if !reflect.DeepEqual(expect, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_ConvertProviderNumber(t *testing.T) {
	tests := []struct {
		name   string
		input  IncomingPhoneNumber
		expect *providernumber.ProviderNumber
	}{
		{"emergency active", IncomingPhoneNumber{SID: "PN1", EmergencyStatus: "Active"}, &providernumber.ProviderNumber{ID: "PN1", Status: number.StatusActive, EmergencyEnabled: true}},
		{"emergency inactive", IncomingPhoneNumber{SID: "PN2", EmergencyStatus: "Inactive"}, &providernumber.ProviderNumber{ID: "PN2", Status: number.StatusActive}},
		{"no emergency field", IncomingPhoneNumber{SID: "PN3"}, &providernumber.ProviderNumber{ID: "PN3", Status: number.StatusActive}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := tt.input.ConvertProviderNumber(); !reflect.DeepEqual(tt.expect, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, res)
			}
		})
	}
}
