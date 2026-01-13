package trunkhandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_EventCUCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer       *cmcustomer.Customer
		responseTrunks []*trunk.Trunk

		expectFilter map[trunk.Field]any
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("e5412000-f09b-11ee-9f74-5b05e21bb6fc"),
			},
			responseTrunks: []*trunk.Trunk{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e5932bfc-f09b-11ee-bdc7-5fdd86be2c03"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e5c9394a-f09b-11ee-b203-439bb4611917"),
					},
				},
			},

			expectFilter: map[trunk.Field]any{
				trunk.FieldCustomerID: uuid.FromStringOrNil("e5412000-f09b-11ee-9f74-5b05e21bb6fc"),
				trunk.FieldDeleted:    false,
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

			h := &trunkHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockDB.EXPECT().TrunkGets(ctx, uint64(1000), gomock.Any(), tt.expectFilter.Return(tt.responseTrunks, nil)

			for _, t := range tt.responseTrunks {

				mockDB.EXPECT().TrunkDelete(ctx, t.ID.Return(nil)
				mockDB.EXPECT().TrunkGet(ctx, t.ID.Return(t, nil)
				mockDB.EXPECT().SIPAuthDelete(ctx, t.ID.Return(nil)
				mockNotify.EXPECT().PublishEvent(ctx, trunk.EventTypeTrunkDeleted, t)
			}

			if err := h.EventCUCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
