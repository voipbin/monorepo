package requesthandler

import (
	reflect "reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"
)

func TestNMAvailableNumbersGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueStorage:   "bin-manager.storage-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
		queueNumber:    "bin-manager.number-manager.request",
	}

	type test struct {
		name string

		userID      uint64
		pageSize    uint64
		countryCode string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult []nmavailablenumber.AvailableNumber
	}

	tests := []test{
		{
			"normal",

			1,
			10,
			"US",

			"bin-manager.number-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/available_numbers?page_size=10&user_id=1&country_code=US",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"number":"+16188850188","provider_name":"telnyx","country":"US","region":"IL","postal_code":"","features":["emergency","fax","voice","sms"],"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
			[]nmavailablenumber.AvailableNumber{
				{
					Number:       "+16188850188",
					ProviderName: "telnyx",
					Country:      "US",
					Region:       "IL",
					Features:     []nmavailablenumber.Feature{"emergency", "fax", "voice", "sms"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.NMAvailableNumbersGet(tt.userID, tt.pageSize, tt.countryCode)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
