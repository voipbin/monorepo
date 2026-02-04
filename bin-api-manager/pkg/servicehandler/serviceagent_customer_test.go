package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ServiceAgentCustomerGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		responseCustomer *cscustomer.Customer
		expectedRes      *cscustomer.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("31cd5e88-b898-11ef-981c-b7b9c42c9e03"),
				},
			},

			responseCustomer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("fd9e1d8c-bc89-11ef-9c59-e39b69889bbf"),

				Name:   "test name",
				Detail: "test detail",

				Email:       "test@voipbin.net",
				PhoneNumber: "+123456789",
				Address:     "somewhere over the rainbow",

				WebhookMethod: cscustomer.WebhookMethodGet,
				WebhookURI:    "test.voipbin.net",

				BillingAccountID: uuid.FromStringOrNil("fe6f4c86-bc89-11ef-a7d5-7bf111ffbe12"),

				TMCreate: "2024-12-01T10:15:30.123456Z",
				TMUpdate: "2024-12-01T10:15:30.123457Z",
				TMDelete: "2024-12-01T10:15:30.123458Z",
			},
			expectedRes: &cscustomer.WebhookMessage{
				ID: uuid.FromStringOrNil("fd9e1d8c-bc89-11ef-9c59-e39b69889bbf"),

				Name:   "test name",
				Detail: "test detail",

				TMCreate: "2024-12-01T10:15:30.123456Z",
				TMUpdate: "2024-12-01T10:15:30.123457Z",
				TMDelete: "2024-12-01T10:15:30.123458Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.agent.CustomerID).Return(tt.responseCustomer, nil)

			res, err := h.ServiceAgentCustomerGet(ctx, tt.agent)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectedRes, res)
			}
		})
	}
}
