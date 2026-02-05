package activeflowhandler

import (
	"context"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_EventCallHangup(t *testing.T) {

	tests := []struct {
		name string

		call               *cmcall.Call
		responseActiveflow *activeflow.Activeflow

		expectUpdateFields map[activeflow.Field]any
	}{
		{
			name: "normal",

			call: &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("43af6682-ecd8-11ee-8a52-639e2d4145e5"),
				},
				ActiveflowID: uuid.FromStringOrNil("442965e0-ecd8-11ee-b7dd-0bad460f1c42"),
			},
			responseActiveflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("442965e0-ecd8-11ee-b7dd-0bad460f1c42"),
				},
			},
			expectUpdateFields: map[activeflow.Field]any{
				activeflow.FieldStatus: activeflow.StatusEnded,
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
			mockDB.EXPECT().ActiveflowGet(ctx, tt.call.ActiveflowID).Return(tt.responseActiveflow, nil)
			switch tt.responseActiveflow.ReferenceType {
			case activeflow.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallHangup(ctx, tt.responseActiveflow.ReferenceID).Return(&cmcall.Call{}, nil)
			}

			mockDB.EXPECT().ActiveflowUpdate(ctx, tt.responseActiveflow.ID, tt.expectUpdateFields).Return(nil)
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

		expectFilters       map[activeflow.Field]any
		responseActiveflows []*activeflow.Activeflow
	}{
		{
			name: "normal",

			customer: &cucustomer.Customer{
				ID: uuid.FromStringOrNil("77812854-ecdf-11ee-85e1-6fea89f3f255"),
			},

			expectFilters: map[activeflow.Field]any{
				activeflow.FieldCustomerID: uuid.FromStringOrNil("77812854-ecdf-11ee-85e1-6fea89f3f255"),
				activeflow.FieldDeleted:    false,
			},
			responseActiveflows: []*activeflow.Activeflow{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("83faabd2-ecdf-11ee-bed3-3f83c1ac3dbd"),
					},
					Status:   activeflow.StatusEnded,
					TMDelete: nil,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8454d30a-ecdf-11ee-939e-c7eac852f244"),
					},
					Status:   activeflow.StatusEnded,
					TMDelete: nil,
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
			mockDB.EXPECT().ActiveflowList(ctx, gomock.Any(), uint64(1000), tt.expectFilters).Return(tt.responseActiveflows, nil)

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
