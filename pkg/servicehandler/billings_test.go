package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
)

func Test_BillingGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseBillingAcounts []bmbilling.Billing
		expectFilters          map[string]string
		expectRes              []*bmbilling.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",

			responseBillingAcounts: []bmbilling.Billing{
				{
					ID: uuid.FromStringOrNil("e3b28042-f566-11ee-baaf-af8732074c75"),
				},
				{
					ID: uuid.FromStringOrNil("e4214112-f566-11ee-8e24-0f932b0506b8"),
				},
			},
			expectFilters: map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},
			expectRes: []*bmbilling.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("e3b28042-f566-11ee-baaf-af8732074c75"),
				},
				{
					ID: uuid.FromStringOrNil("e4214112-f566-11ee-8e24-0f932b0506b8"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().BillingV1BillingGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseBillingAcounts, nil)

			res, err := h.BillingGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
