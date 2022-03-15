package listenhandler

import (
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/numberhandler"
)

func TestProcessV1AvailableNumbersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockNumber := numberhandler.NewMockNumberHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		numberHandler: mockNumber,
	}

	tests := []struct {
		name        string
		countryCode string
		pageSize    uint
		numbers     []*availablenumber.AvailableNumber

		request  *rabbitmqhandler.Request
		response *rabbitmqhandler.Response
	}{
		{
			"empty numbers",
			"US",
			1,
			[]*availablenumber.AvailableNumber{},
			&rabbitmqhandler.Request{
				URI:    "/v1/available_numbers?country_code=US&page_size=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
		{
			"1 number entry",
			"US",
			1,
			[]*availablenumber.AvailableNumber{
				{
					Number:       "+16188850188",
					ProviderName: number.ProviderNameTelnyx,
					Country:      "US",
					Region:       "IL",
					Features: []availablenumber.Feature{
						availablenumber.FeatureEmergency, availablenumber.FeatureFax, availablenumber.FeatureVoice, availablenumber.FeatureSMS,
					},
				},
			},
			&rabbitmqhandler.Request{
				URI:    "/v1/available_numbers?country_code=US&page_size=1",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"number":"+16188850188","provider_name":"telnyx","country":"US","region":"IL","postal_code":"","features":["emergency","fax","voice","sms"],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockNumber.EXPECT().GetAvailableNumbers(tt.countryCode, tt.pageSize).Return(tt.numbers, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}
