package telnyx

import (
	"reflect"
	"testing"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/providernumber"
)

func Test_ConvertProviderNumber(t *testing.T) {

	tests := []struct {
		name string

		phoneNumber *PhoneNumber

		expectRes *providernumber.ProviderNumber
	}{
		{
			name: "t38 enabled and emergency disabled",

			phoneNumber: &PhoneNumber{
				ID:                   "provider-id-1",
				Status:               PhoneNumberStatusActive,
				T38FaxGatewayEnabled: true,
				EmergencyEnabled:     false,
			},

			expectRes: &providernumber.ProviderNumber{
				ID:               "provider-id-1",
				Status:           number.StatusActive,
				T38Enabled:       true,
				EmergencyEnabled: false,
			},
		},
		{
			name: "t38 disabled and emergency enabled",

			phoneNumber: &PhoneNumber{
				ID:                   "provider-id-2",
				Status:               PhoneNumberStatusActive,
				T38FaxGatewayEnabled: false,
				EmergencyEnabled:     true,
			},

			expectRes: &providernumber.ProviderNumber{
				ID:               "provider-id-2",
				Status:           number.StatusActive,
				T38Enabled:       false,
				EmergencyEnabled: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.phoneNumber.ConvertProviderNumber()
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
