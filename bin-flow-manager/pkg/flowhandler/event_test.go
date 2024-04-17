package flowhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_EventCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer      *cmcustomer.Customer
		responseFlows []*flow.Flow

		expectFilter map[string]string
	}{
		{
			name: "normal",

			customer: &cmcustomer.Customer{
				ID: uuid.FromStringOrNil("264d2078-ecd5-11ee-8120-e31f06a93e99"),
			},
			responseFlows: []*flow.Flow{
				{
					ID: uuid.FromStringOrNil("26c93460-ecd5-11ee-b1ff-d7a1c55584d6"),
				},
				{
					ID: uuid.FromStringOrNil("26fe9f7e-ecd5-11ee-828f-b362008cff0c"),
				},
			},

			expectFilter: map[string]string{
				"customer_id": "264d2078-ecd5-11ee-8120-e31f06a93e99",
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

			h := &flowHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				util:          mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().FlowGets(ctx, gomock.Any(), uint64(1000), tt.expectFilter).Return(tt.responseFlows, nil)

			for _, f := range tt.responseFlows {
				mockDB.EXPECT().FlowDelete(ctx, f.ID).Return(nil)
				mockDB.EXPECT().FlowGet(ctx, f.ID).Return(f, nil)
				mockNotify.EXPECT().PublishEvent(ctx, flow.EventTypeFlowDeleted, f)
			}

			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
