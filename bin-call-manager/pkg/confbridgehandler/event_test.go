package confbridgehandler

import (
	"context"
	"testing"

	"monorepo/bin-call-manager/models/confbridge"
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

		customer            *cmcustomer.Customer
		responseConfbridges []*confbridge.Confbridge

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("428dca6e-f0cf-11ee-8695-935c2a4b9b61"),
			},
			responseConfbridges: []*confbridge.Confbridge{
				{
					ID:       uuid.FromStringOrNil("42e3a4e8-f0cf-11ee-8d63-9748a0e7a936"),
					Status:   confbridge.StatusTerminated,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("43207666-f0cf-11ee-a8a2-1f40b0249a54"),
					Status:   confbridge.StatusTerminated,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilter: map[string]string{
				"customer_id": "428dca6e-f0cf-11ee-8695-935c2a4b9b61",
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

			h := &confbridgeHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConfbridgeGets(ctx, uint64(1000), "", tt.expectFilter).Return(tt.responseConfbridges, nil)

			// delete each calls
			for _, cf := range tt.responseConfbridges {
				mockDB.EXPECT().ConfbridgeGet(ctx, cf.ID).Return(cf, nil)

				// dbDelete
				mockDB.EXPECT().ConfbridgeDelete(ctx, cf.ID).Return(nil)
				mockDB.EXPECT().ConfbridgeGet(ctx, cf.ID).Return(cf, nil)
				mockNotify.EXPECT().PublishEvent(ctx, confbridge.EventTypeConfbridgeDeleted, cf)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
