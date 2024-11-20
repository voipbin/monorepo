package groupcallhandler

import (
	"context"
	"testing"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer           *cmcustomer.Customer
		responseGroupcalls []*groupcall.Groupcall

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("bfbc81b8-f0c7-11ee-b74f-a3f6b95bc57e"),
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c007ba2a-f0c7-11ee-857b-777b125077ae"),
					},
					Status:   groupcall.StatusHangup,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0404f8e-f0c7-11ee-8e1d-df676cb88c41"),
					},
					Status:   groupcall.StatusHangup,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
			},

			expectFilter: map[string]string{
				"customer_id": "bfbc81b8-f0c7-11ee-b74f-a3f6b95bc57e",
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

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().GroupcallGets(ctx, uint64(1000), "", tt.expectFilter).Return(tt.responseGroupcalls, nil)

			// delete each groupcalls
			for _, gc := range tt.responseGroupcalls {
				mockDB.EXPECT().GroupcallGet(ctx, gc.ID).Return(gc, nil)

				// dbDelete
				mockDB.EXPECT().GroupcallDelete(ctx, gc.ID).Return(nil)
				mockDB.EXPECT().GroupcallGet(ctx, gc.ID).Return(gc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gc.CustomerID, groupcall.EventTypeGroupcallDeleted, gc)

			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
