package requesthandler

import (
	"context"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_BillingV1BillingGets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseBillings *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     []bmbilling.Billing
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-06-08 03:22:17.995000",
			filters: map[string]string{
				"customer_id": "84ec5606-f556-11ee-b9a0-dbdcc291145b",
			},

			responseBillings: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"854608c2-f556-11ee-bcaa-7b93c058e8f6"},{"id":"85fdae46-f556-11ee-ba13-c3b959ad9a23"}]`),
			},

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:    "/v1/billings?page_token=2023-06-08+03%3A22%3A17.995000&page_size=10&filter_customer_id=84ec5606-f556-11ee-b9a0-dbdcc291145b",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			expectRes: []bmbilling.Billing{
				{
					ID: uuid.FromStringOrNil("854608c2-f556-11ee-bcaa-7b93c058e8f6"),
				},
				{
					ID: uuid.FromStringOrNil("85fdae46-f556-11ee-ba13-c3b959ad9a23"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseBillings, nil)

			res, err := reqHandler.BillingV1BillingGets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
