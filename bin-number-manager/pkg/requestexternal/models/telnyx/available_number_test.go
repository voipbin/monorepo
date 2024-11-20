package telnyx

import (
	reflect "reflect"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
)

func TestConvertAvailableNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name   string
		number *AvailableNumber

		expectRes *availablenumber.AvailableNumber
	}

	tests := []test{
		{
			"normal",
			&AvailableNumber{
				PhoneNumber:  "+16188850188",
				Reservable:   true,
				QuickShip:    true,
				VanityFormat: "",
				RecordType:   "available_phone_number",
				CostInformation: AvailableCostInformation{
					MonthlyCost: "1.00000",
					UpfrontCost: "1.00000",
					Currency:    "USD",
				},
				Features: []AvailableFeature{
					{
						Name: "emergency",
					},
					{
						Name: "fax",
					},
					{
						Name: "voice",
					},
					{
						Name: "sms",
					},
				},
				RegionInformation: []AvailableRegionInformation{
					{
						RegionName: "IL",
						RegionType: "state",
					},
					{
						RegionName: "US",
						RegionType: "country_code",
					},
					{
						RegionName: "DOW",
						RegionType: "rate_center",
					},
				},
			},
			&availablenumber.AvailableNumber{
				Number:       "+16188850188",
				ProviderName: number.ProviderNameTelnyx,

				Country: "US",
				Region:  "IL",
				Features: []availablenumber.Feature{
					availablenumber.FeatureEmergency, availablenumber.FeatureFax, availablenumber.FeatureVoice, availablenumber.FeatureSMS,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.number.ConvertAvailableNumber()
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}
