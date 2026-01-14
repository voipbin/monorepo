package requesthandler

import (
	"context"
	reflect "reflect"
	"testing"

	nmavailablenumber "monorepo/bin-number-manager/models/availablenumber"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_NumberV1AvailableNumberGets(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		pageSize    uint64
		countryCode string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectResult []nmavailablenumber.AvailableNumber
	}{
		{
			"normal",

			uuid.FromStringOrNil("b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"),
			10,
			"US",

			"bin-manager.number-manager.request",
			&sock.Request{
				URI:      "/v1/available_numbers?page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			Data:     []byte(`{"country_code":"US","customer_id":"b7041f62-7ff5-11ec-b1dd-d7e05b3c5096"}`),
			},
			&sock.Response{
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
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			filters := map[string]any{
				"customer_id": tt.customerID,
				"country_code": tt.countryCode,
			}
			res, err := reqHandler.NumberV1AvailableNumberGets(ctx, tt.pageSize, filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
