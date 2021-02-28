package numberhandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler/models/telnyx"
)

func TestGetAvailableNumbers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewNumberHandler(mockReq, mockDB, mockCache)

	type test struct {
		name    string
		country string
		limit   uint

		numbers   []*telnyx.AvailableNumber
		expectRes []*models.AvailableNumber
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
			[]*models.AvailableNumber{
				{
					Number:       "+16188850188",
					ProviderName: models.NumberProviderNameTelnyx,

					Country: "US",
					Region:  "IL",
					Features: []models.AvailableNumberFeature{
						models.AvailableNumberFeatureEmergency, models.AvailableNumberFeatureFax, models.AvailableNumberFeatureVoice, models.AvailableNumberFeatureSMS,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().TelnyxAvailableNumberGets(tt.country, "", "", tt.limit).Return(tt.numbers, nil)

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
