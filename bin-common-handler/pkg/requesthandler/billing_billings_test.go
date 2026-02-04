package requesthandler

import (
	"context"
	"reflect"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_BillingV1BillingList(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[bmbilling.Field]any

		responseBillings *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []bmbilling.Billing
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-06-08T03:22:17.995000Z",
			filters: map[bmbilling.Field]any{
				bmbilling.FieldCustomerID: uuid.FromStringOrNil("84ec5606-f556-11ee-b9a0-dbdcc291145b"),
			},

			responseBillings: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"854608c2-f556-11ee-bcaa-7b93c058e8f6"},{"id":"85fdae46-f556-11ee-ba13-c3b959ad9a23"}]`),
			},

			expectTarget: "bin-manager.billing-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/billings?page_token=2023-06-08T03%3A22%3A17.995000Z&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"84ec5606-f556-11ee-b9a0-dbdcc291145b"}`),
			},
			expectRes: []bmbilling.Billing{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("854608c2-f556-11ee-bcaa-7b93c058e8f6"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("85fdae46-f556-11ee-ba13-c3b959ad9a23"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.responseBillings, nil)

			res, err := h.BillingV1BillingList(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
