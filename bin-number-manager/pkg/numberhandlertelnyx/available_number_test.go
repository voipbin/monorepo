package numberhandlertelnyx

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-number-manager/models/availablenumber"
	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/pkg/requestexternal"
	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
)

func TestGetAvailableNumbers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := numberHandlerTelnyx{
		requestExternal: mockExternal,
	}

	type test struct {
		name    string
		country string
		limit   uint

		numbers   []*telnyx.AvailableNumber
		expectRes []*availablenumber.AvailableNumber
	}

	tests := []test{
		{
			"normal us",
			"us",
			1,
			[]*telnyx.AvailableNumber{
				{
					PhoneNumber:  "+16188850188",
					Reservable:   true,
					QuickShip:    true,
					VanityFormat: "",
					RecordType:   "available_phone_number",
					CostInformation: telnyx.AvailableCostInformation{
						MonthlyCost: "1.00000",
						UpfrontCost: "1.00000",
						Currency:    "USD",
					},
					Features: []telnyx.AvailableFeature{
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
					RegionInformation: []telnyx.AvailableRegionInformation{
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
			},
			[]*availablenumber.AvailableNumber{
				{
					Number:       "+16188850188",
					ProviderName: number.ProviderNameTelnyx,

					Country: "US",
					Region:  "IL",
					Features: []availablenumber.Feature{
						availablenumber.FeatureEmergency, availablenumber.FeatureFax, availablenumber.FeatureVoice, availablenumber.FeatureSMS,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockExternal.EXPECT().TelnyxAvailableNumberGets(defaultToken, tt.country, "", "", tt.limit).Return(tt.numbers, nil)

			res, err := h.GetAvailableNumbers(tt.country, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
