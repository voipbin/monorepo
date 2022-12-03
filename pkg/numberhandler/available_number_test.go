package numberhandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandlertelnyx"
)

func TestGetAvailableNumbersTelnyx(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockTelnyx := numberhandlertelnyx.NewMockNumberHandlerTelnyx(mc)

	h := numberHandler{
		reqHandler:          mockReq,
		db:                  mockDB,
		numberHandlerTelnyx: mockTelnyx,
	}

	type test struct {
		name    string
		country string
		limit   uint

		expectRes []*availablenumber.AvailableNumber
	}

	tests := []test{
		{
			"normal us",
			"us",
			1,
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

			mockTelnyx.EXPECT().GetAvailableNumbers(tt.country, tt.limit).Return(tt.expectRes, nil)

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
