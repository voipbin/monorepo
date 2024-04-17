package servicehandler

import (
	"context"
	"reflect"
	"testing"

	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_GroupcallGet(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			"normal",

			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425"),

			&cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			&cmgroupcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("979b59ec-bf00-11ed-a60e-77087af74425"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			res, err := h.GroupcallGet(ctx, tt.agent, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
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

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
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
				ID:       uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				TMDelete: defaultTimestamp,
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID:       uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				TMDelete: defaultTimestamp,
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
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.agent.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(tt.responseFlow, nil)
			}
			mockReq.EXPECT().CallV1GroupcallCreate(ctx, uuid.Nil, tt.agent.CustomerID, targetFlowID, tt.source, tt.destinations, uuid.Nil, uuid.Nil, tt.ringMethod, tt.answerMethod).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallCreate(ctx, tt.agent, tt.source, tt.destinations, tt.flowID, tt.actions, tt.ringMethod, tt.answerMethod)
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

		agent       *amagent.Agent
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			groupcallID: uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),

			responseGroupcall: &cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("415fd3ee-bf02-11ed-90a4-1fde392a001c"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockReq.EXPECT().CallV1GroupcallHangup(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallHangup(ctx, tt.agent, tt.groupcallID)
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

		agent       *amagent.Agent
		groupcallID uuid.UUID

		responseGroupcall *cmgroupcall.Groupcall
		expectRes         *cmgroupcall.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			groupcallID: uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),

			responseGroupcall: &cmgroupcall.Groupcall{
				ID:         uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &cmgroupcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("3c3b27dc-bf03-11ed-9885-d7004ea1cd6a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1GroupcallGet(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)
			mockReq.EXPECT().CallV1GroupcallDelete(ctx, tt.groupcallID).Return(tt.responseGroupcall, nil)

			res, err := h.GroupcallDelete(ctx, tt.agent, tt.groupcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
