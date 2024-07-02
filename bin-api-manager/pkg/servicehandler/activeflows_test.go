package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_activeflowGet(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f1d53156-8dec-11ee-98a0-6ba69fe98bd2"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			activeflowID: uuid.FromStringOrNil("306d40a4-cb22-11ed-a796-4776eeb9578e"),

			responseActiveflow: &fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("306d40a4-cb22-11ed-a796-4776eeb9578e"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.activeflowGet(ctx, tt.agent, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseActiveflow) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.responseActiveflow, res)
			}
		})
	}
}

func Test_ActiveflowCreate(t *testing.T) {

	tests := []struct {
		name string

		agent        *amagent.Agent
		activeflowID uuid.UUID
		flowID       uuid.UUID
		actions      []fmaction.Action

		responseUUID       uuid.UUID
		responseFlow       *fmflow.Flow
		responseActiveflow *fmactiveflow.Activeflow

		expectRes *fmactiveflow.WebhookMessage
	}{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			activeflowID: uuid.FromStringOrNil("2498ab92-c824-11ee-8470-b77066a63403"),
			flowID:       uuid.FromStringOrNil("24e16bd4-c824-11ee-8e8f-ef99de05a30a"),
			actions:      []fmaction.Action{},

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("24e16bd4-c824-11ee-8e8f-ef99de05a30a"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("2498ab92-c824-11ee-8470-b77066a63403"),
			},

			expectRes: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("2498ab92-c824-11ee-8470-b77066a63403"),
			},
		},
		{
			name: "has no activeflow id",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			activeflowID: uuid.Nil,
			flowID:       uuid.FromStringOrNil("de52be1c-c827-11ee-b844-2bf5469a5b7f"),
			actions:      []fmaction.Action{},

			responseUUID: uuid.FromStringOrNil("c9de6aa8-c827-11ee-afc6-9fa0a350a342"),
			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("de52be1c-c827-11ee-b844-2bf5469a5b7f"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("c9de6aa8-c827-11ee-afc6-9fa0a350a342"),
			},

			expectRes: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("c9de6aa8-c827-11ee-afc6-9fa0a350a342"),
			},
		},
		{
			name: "has no flow id but has actions",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			activeflowID: uuid.FromStringOrNil("94795dac-c82a-11ee-82c3-2b67b1791a1b"),
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			responseFlow: &fmflow.Flow{
				ID:         uuid.FromStringOrNil("949fc9a6-c82a-11ee-99f7-5be67e385f05"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				TMDelete:   defaultTimestamp,
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("94795dac-c82a-11ee-82c3-2b67b1791a1b"),
			},

			expectRes: &fmactiveflow.WebhookMessage{
				ID: uuid.FromStringOrNil("94795dac-c82a-11ee-82c3-2b67b1791a1b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			activeflowID := tt.activeflowID
			if activeflowID == uuid.Nil {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
				activeflowID = tt.responseUUID
			}

			flowID := tt.flowID
			if tt.flowID == uuid.Nil {
				mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.agent.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.actions, false).Return(tt.responseFlow, nil)
				flowID = tt.responseFlow.ID
			}
			mockReq.EXPECT().FlowV1FlowGet(ctx, flowID).Return(tt.responseFlow, nil)

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, activeflowID, flowID, fmactiveflow.ReferenceTypeNone, uuid.Nil).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Return(nil)

			res, err := h.ActiveflowCreate(ctx, tt.agent, tt.activeflowID, tt.flowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowGet(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		activeflowID uuid.UUID

		responseActiveflow *fmactiveflow.Activeflow
		expectRes          *fmactiveflow.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f20e009e-8dec-11ee-80ed-df2de3ed9cb4"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			activeflowID: uuid.FromStringOrNil("f236da96-8dec-11ee-a3c2-d786fe7eaaae"),

			responseActiveflow: &fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("f236da96-8dec-11ee-a3c2-d786fe7eaaae"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},
			expectRes: &fmactiveflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("f236da96-8dec-11ee-a3c2-d786fe7eaaae"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.ActiveflowGet(ctx, tt.agent, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseActiveflows []fmactiveflow.Activeflow
		expectFilters       map[string]string
		expectRes           []*fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("040422b6-3771-11ed-801b-27518c703c82"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]fmactiveflow.Activeflow{
				{
					ID: uuid.FromStringOrNil("23dc5a36-cb23-11ed-8a25-8f48bd8c19bf"),
				},
			},
			map[string]string{
				"customer_id": "1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3",
				"deleted":     "false",
			},
			[]*fmactiveflow.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("23dc5a36-cb23-11ed-8a25-8f48bd8c19bf"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowGets(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseActiveflows, nil)

			res, err := h.ActiveflowGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowStop(t *testing.T) {

	tests := []struct {
		name               string
		agent              *amagent.Agent
		activeflowID       uuid.UUID
		responseActiveflow *fmactiveflow.Activeflow

		expectRes *fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
			&fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&fmactiveflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("2b4b10f4-cb24-11ed-ad87-0fe018a49bcd"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowStop(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.ActiveflowStop(ctx, tt.agent, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ActiveflowDelete(t *testing.T) {

	tests := []struct {
		name               string
		agent              *amagent.Agent
		activeflowID       uuid.UUID
		responseActiveflow *fmactiveflow.Activeflow

		expectRes *fmactiveflow.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("14003656-8e5e-11ee-b952-0ff7940c8c0e"),
					CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
			&fmactiveflow.Activeflow{
				ID:         uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
				TMDelete:   defaultTimestamp,
			},

			&fmactiveflow.WebhookMessage{
				ID:         uuid.FromStringOrNil("73161b68-cb24-11ed-8253-2f25bfb9d81b"),
				CustomerID: uuid.FromStringOrNil("1ed3b04a-7ffa-11ec-a974-cbbe9a9538b3"),
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

			mockReq.EXPECT().FlowV1ActiveflowGet(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)
			mockReq.EXPECT().FlowV1ActiveflowDelete(ctx, tt.activeflowID).Return(tt.responseActiveflow, nil)

			res, err := h.ActiveflowDelete(ctx, tt.agent, tt.activeflowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}
