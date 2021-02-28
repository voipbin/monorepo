package telnyx

import (
	reflect "reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
)

func TestConvertAvailableNumber(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	type test struct {
		name   string
		number *AvailableNumber

		expectRes *models.AvailableNumber
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
			&models.AvailableNumber{
				Number:       "+16188850188",
				ProviderName: models.NumberProviderNameTelnyx,

				Country: "US",
				Region:  "IL",
				Features: []models.AvailableNumberFeature{
					models.AvailableNumberFeatureEmergency, models.AvailableNumberFeatureFax, models.AvailableNumberFeatureVoice, models.AvailableNumberFeatureSMS,
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
