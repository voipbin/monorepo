package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_groupcallGet(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
	}{
		{
			"normal",

			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("975670c0-bf00-11ed-b8d7-8b8f0c7d3a15"),
			},
			uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425"),

			&cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425"),
				CustomerID: uuid.FromStringOrNil("975670c0-bf00-11ed-b8d7-8b8f0c7d3a15"),
				TMDelete:   defaultTimestamp,
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

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)

			res, err := h.groupcallGet(ctx, tt.customer, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseGroupcall) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseGroupcall, res)
			}
		})
	}
}

func Test_GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		customer     *cscustomer.Customer
		source       commonaddress.Address
		destinations []commonaddress.Address
		flowID       uuid.UUID
		actions      []fmaction.Action
		ringMethod   cmgroupcall.RingMethod
		answerMethod cmgroupcall.AnswerMethod

		responseFlow      *fmflow.Flow
		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
			},
			source: commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "testsource@test.com",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeSIP,
					Target: "testdestination@test.com",
				},
			},
			flowID:       uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			actions:      []fmaction.Action{},
			ringMethod:   cmgroupcall.RingMethodRingAll,
			answerMethod: cmgroupcall.AnswerMethodHangupOthers,

			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("afce8664-bf01-11ed-b58c-dbe9035888fa"),
			},
			responseGroupcall: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID: uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
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

			targetFlowID := tt.flowID
			if targetFlowID == uuid.Nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customer.ID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(tt.responseFlow, nil)
			}
			mockReq.EXPECT().CallV1GroupcallCreate(ctx, tt.customer.ID, tt.source, tt.destinations, targetFlowID, uuid.Nil, tt.ringMethod, tt.answerMethod, false).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallCreate(ctx, tt.customer, tt.source, tt.destinations, tt.flowID, tt.actions, tt.ringMethod, tt.answerMethod)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallHangup(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("411e627e-bf02-11ed-adb4-1b6252b5f9df"),
			},
			groupcallID: uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),

			responseGroupcall: &cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),
				CustomerID: uuid.FromStringOrNil("411e627e-bf02-11ed-adb4-1b6252b5f9df"),
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),
				CustomerID: uuid.FromStringOrNil("411e627e-bf02-11ed-adb4-1b6252b5f9df"),
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

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockReq.EXPECT().CallV1GroupcallHangup(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallHangup(ctx, tt.customer, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallDelete(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("3c623ec6-bf03-11ed-a301-6705f8521dae"),
			},
			groupcallID: uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),

			responseGroupcall: &cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),
				CustomerID: uuid.FromStringOrNil("3c623ec6-bf03-11ed-a301-6705f8521dae"),
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),
				CustomerID: uuid.FromStringOrNil("3c623ec6-bf03-11ed-a301-6705f8521dae"),
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

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockReq.EXPECT().CallV1GroupcallDelete(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallDelete(ctx, tt.customer, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
