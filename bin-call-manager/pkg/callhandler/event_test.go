package callhandler

import (
	"context"
	"testing"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer      *cmcustomer.Customer
		responseCalls []*call.Call

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3"),
			},
			responseCalls: []*call.Call{
				{
					ID:       uuid.FromStringOrNil("8c70ee42-f0c3-11ee-b8d2-b3b3892bc551"),
					Status:   call.StatusHangup,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("8c9af3f4-f0c3-11ee-9351-cfa1330e7d25"),
					Status:   call.StatusHangup,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilter: map[string]string{
				"customer_id": "8c0daf80-f0c3-11ee-9ed5-6b65132a6fc3",
				"deleted":     "false",
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

			mockDB.EXPECT().CallGets(ctx, uint64(1000), "", tt.expectFilter).Return(tt.responseCalls, nil)

			// delete each calls
			for _, c := range tt.responseCalls {
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)

				// dbDelete
				mockDB.EXPECT().CallDelete(ctx, c.ID).Return(nil)
				mockDB.EXPECT().CallGet(ctx, c.ID).Return(c, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, c.CustomerID, call.EventTypeCallDeleted, c)

			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
