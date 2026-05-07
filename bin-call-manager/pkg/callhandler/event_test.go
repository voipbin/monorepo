package callhandler

import (
	"context"
	"testing"

	"monorepo/bin-call-manager/models/call"
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	customerID := uuid.FromStringOrNil("8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3")
	configID := uuid.FromStringOrNil("a1000000-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		customer        *cmcustomer.Customer
		responseCalls   []*call.Call
		responseConfig  *outboundconfig.OutboundConfig

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: customerID,
			},
			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c70ee42-f0c3-11ee-b8d2-b3b3892bc551"),
					},
					Status:   call.StatusHangup,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c9af3f4-f0c3-11ee-9351-cfa1330e7d25"),
					},
					Status:   call.StatusHangup,
					TMDelete: nil,
				},
			},
			responseConfig: &outboundconfig.OutboundConfig{
				ID:         configID,
				CustomerID: customerID,
			},

			expectFilter: map[string]string{
				"customer_id": "8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3",
				"deleted":     "false",
			},
		},
		{
			name: "no outbound config",

			customer: &cmcustomer.Customer{
				ID: customerID,
			},
			responseCalls:  []*call.Call{},
			responseConfig: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &callHandler{
				reqHandler:            mockReq,
				db:                    mockDB,
				notifyHandler:         mockNotify,
				utilHandler:           mockUtil,
				outboundConfigHandler: mockOutboundConfig,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(tt.responseCalls, nil)

			// delete each call
			for _, c := range tt.responseCalls {
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
				mockDB.EXPECT().CallDelete(ctx, c.ID).Return(nil)
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallDeleted, c)
			}

			// delete outbound config
			mockOutboundConfig.EXPECT().GetByCustomerID(ctx, tt.customer.ID).Return(tt.responseConfig, nil)
			if tt.responseConfig != nil {
				mockOutboundConfig.EXPECT().Delete(ctx, tt.responseConfig.ID).Return(tt.responseConfig, nil)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCUCustomerFrozen(t *testing.T) {

	tests := []struct {
		name string

		customer      *cmcustomer.Customer
		responseCalls []*call.Call
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3"),
			},
			responseCalls: []*call.Call{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c70ee42-f0c3-11ee-b8d2-b3b3892bc551"),
					},
					Status: call.StatusTerminating,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8c9af3f4-f0c3-11ee-9351-cfa1330e7d25"),
					},
					Status: call.StatusTerminating,
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallList(ctx, uint64(1000), "", gomock.Any()).Return(tt.responseCalls, nil)

			// hangup each call - HangingUp calls Get, then returns early because status is already terminating
			for _, c := range tt.responseCalls {
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
			}

			if err := h.EventCUCustomerFrozen(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
