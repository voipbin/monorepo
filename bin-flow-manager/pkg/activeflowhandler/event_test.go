package activeflowhandler

import (
	"context"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_EventCallHangup(t *testing.T) {

	tests := []struct {
		name string

		call               *cmcall.Call
		responseActiveflow *activeflow.Activeflow
	}{
		{
			name: "normal",

			call: &cmcall.Call{
				ID:           uuid.FromStringOrNil("43af6682-ecd8-11ee-8a52-639e2d4145e5"),
				ActiveFlowID: uuid.FromStringOrNil("442965e0-ecd8-11ee-b7dd-0bad460f1c42"),
			},
			responseActiveflow: &activeflow.Activeflow{
				ID: uuid.FromStringOrNil("442965e0-ecd8-11ee-b7dd-0bad460f1c42"),
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

			h := &activeflowHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			// stop()
			mockDB.EXPECT().ActiveflowGet(ctx, tt.call.ActiveFlowID).Return(tt.responseActiveflow, nil)
			switch tt.responseActiveflow.ReferenceType {
			case activeflow.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallHangup(ctx, tt.responseActiveflow.ReferenceID).Return(&cmcall.Call{}, nil)
			}

			mockDB.EXPECT().ActiveflowSetStatus(ctx, tt.responseActiveflow.ID, activeflow.StatusEnded).Return(nil)
			mockDB.EXPECT().ActiveflowGet(ctx, tt.responseActiveflow.ID).Return(tt.responseActiveflow, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseActiveflow.CustomerID, activeflow.EventTypeActiveflowUpdated, tt.responseActiveflow)

			if err := h.EventCallHangup(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_EventCustomerDeleted(t *testing.T) {

	tests := []struct {
		name string

		customer *cucustomer.Customer

		expectFilters       map[string]string
		responseActiveflows []*activeflow.Activeflow
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("77812854-ecdf-11ee-85e1-6fea89f3f255"),
			},

			expectFilters: map[string]string{
				"customer_id": "77812854-ecdf-11ee-85e1-6fea89f3f255",
				"deleted":     "false",
			},
			responseActiveflows: []*activeflow.Activeflow{
				{
					ID:       uuid.FromStringOrNil("83faabd2-ecdf-11ee-bed3-3f83c1ac3dbd"),
					Status:   activeflow.StatusEnded,
					TMDelete: dbhandler.DefaultTimeStamp,
				},
				{
					ID:       uuid.FromStringOrNil("8454d30a-ecdf-11ee-939e-c7eac852f244"),
					Status:   activeflow.StatusEnded,
					TMDelete: dbhandler.DefaultTimeStamp,
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

			h := &activeflowHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockDB.EXPECT().ActiveflowGets(ctx, gomock.Any(), uint64(1000), tt.expectFilters).Return(tt.responseActiveflows, nil)

			// delete
			for _, af := range tt.responseActiveflows {
				mockDB.EXPECT().ActiveflowGet(ctx, af.ID).Return(af, nil)

				mockDB.EXPECT().ActiveflowDelete(ctx, af.ID).Return(nil)
				mockDB.EXPECT().ActiveflowGet(ctx, af.ID).Return(af, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowDeleted, af)
			}

			if err := h.EventCustomerDeleted(ctx, tt.customer); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
