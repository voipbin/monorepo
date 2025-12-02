package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_FlowCreate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		flowName string
		detail   string
		actions  []fmaction.Action
		persist  bool

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
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

			flowName: "test name",
			detail:   "test detail",
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			persist: true,

			response: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50daef5a-f2f6-11ea-9649-33c2eb34ec4c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.agent.CustomerID, fmflow.TypeFlow, tt.flowName, tt.detail, tt.actions, uuid.Nil, tt.persist).Return(tt.response, nil)
			res, err := h.FlowCreate(ctx, tt.agent, tt.flowName, tt.detail, tt.actions, tt.persist)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowDelete(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		flowID uuid.UUID

		responseFlow *fmflow.Flow
		expectRes    *fmflow.WebhookMessage
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
			flowID: uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),

			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:     "test",
				Detail:   "test detail",
				Actions:  []fmaction.Action{},
				TMDelete: defaultTimestamp,
			},
			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00efc020-67cb-11eb-bd5e-b3c491185912"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:     "test",
				Detail:   "test detail",
				Actions:  []fmaction.Action{},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			mockReq.EXPECT().FlowV1FlowDelete(ctx, tt.flowID).Return(tt.responseFlow, nil)

			res, err := h.FlowDelete(ctx, tt.agent, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowGet(t *testing.T) {

	tests := []struct {
		name   string
		agent  *amagent.Agent
		flowID uuid.UUID

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
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
			flowID: uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),

			response: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:     "test",
				Detail:   "test detail",
				Actions:  []fmaction.Action{},
				TMDelete: defaultTimestamp,
			},
			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1f80baf0-0c5c-11eb-9df4-1f217b30d87c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:     "test",
				Detail:   "test detail",
				Actions:  []fmaction.Action{},
				TMDelete: defaultTimestamp,
			},
		},
		{
			name: "action answer",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			flowID: uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),

			response: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:   "test",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("61f86f60-66af-11eb-917f-838fd6836e1f"),
						Type: fmaction.TypeAnswer,
					},
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5ce8210a-66af-11eb-a7f4-a36a8393fce1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:   "test",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("61f86f60-66af-11eb-917f-838fd6836e1f"),
						Type: "answer",
					},
				},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)

			res, err := h.FlowGet(ctx, tt.agent, tt.flowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowGets(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseFlows []fmflow.Flow
		expectFilters map[fmflow.Field]any
		expectRes     []*fmflow.WebhookMessage
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
			pageToken: "2020-10-20T01:00:00.995000",
			pageSize:  10,

			responseFlows: []fmflow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:     "test1",
					Detail:   "test detail1",
					Actions:  []fmaction.Action{},
					TMDelete: defaultTimestamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:     "test2",
					Detail:   "test detail2",
					Actions:  []fmaction.Action{},
					TMDelete: defaultTimestamp,
				},
			},
			expectFilters: map[fmflow.Field]any{
				fmflow.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				fmflow.FieldType:       fmflow.TypeFlow,
				fmflow.FieldDeleted:    false,
			},
			expectRes: []*fmflow.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:     "test1",
					Detail:   "test detail1",
					Actions:  []fmaction.Action{},
					TMDelete: defaultTimestamp,
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:     "test2",
					Detail:   "test detail2",
					Actions:  []fmaction.Action{},
					TMDelete: defaultTimestamp,
				},
			},
		},
		{
			name: "1 action",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "2020-10-20T01:00:00.995000",
			pageSize:  10,

			responseFlows: []fmflow.Flow{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:   "test1",
					Detail: "test detail1",
					Actions: []fmaction.Action{
						{
							ID:   uuid.FromStringOrNil("775f5cde-66ae-11eb-9626-0f488d332e1e"),
							Type: fmaction.TypeAnswer,
						},
					},
					TMDelete: defaultTimestamp,
				},
			},
			expectFilters: map[fmflow.Field]any{
				fmflow.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				fmflow.FieldDeleted:    false,
				fmflow.FieldType:       fmflow.TypeFlow,
			},
			expectRes: []*fmflow.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("5a109d00-66ae-11eb-ad00-bbcf73569888"),
						CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
					},
					Name:   "test1",
					Detail: "test detail1",
					Actions: []fmaction.Action{
						{
							ID:   uuid.FromStringOrNil("775f5cde-66ae-11eb-9626-0f488d332e1e"),
							Type: "answer",
						},
					},
					TMDelete: defaultTimestamp,
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseFlows, nil)

			res, err := h.FlowGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowUpdate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		flowID   uuid.UUID
		flowName string
		detail   string
		actions  []fmaction.Action

		responseFlow *fmflow.Flow
		expectRes    *fmflow.WebhookMessage
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

			flowID:   uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
			flowName: "test name",
			detail:   "test detail",
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},
			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a64ff8ce-1ab3-4564-9d34-e5f3147810e5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			f := &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: tt.flowID,
				},
				Name:    tt.flowName,
				Detail:  tt.detail,
				Actions: tt.actions,
			}

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			mockReq.EXPECT().FlowV1FlowUpdate(ctx, f).Return(tt.responseFlow, nil)
			res, err := h.FlowUpdate(ctx, tt.agent, f)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_FlowUpdateActions(t *testing.T) {

	tests := []struct {
		name    string
		agent   *amagent.Agent
		flowID  uuid.UUID
		actions []fmaction.Action

		response  *fmflow.Flow
		expectRes *fmflow.WebhookMessage
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
			flowID: uuid.FromStringOrNil("1058806a-45c1-4bc0-9605-1148e20008c1"),
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			response: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				TMDelete: defaultTimestamp,
			},

			expectRes: &fmflow.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("00498856-678d-11eb-89a6-37bc9314dc94"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
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

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.response, nil)
			mockReq.EXPECT().FlowV1FlowUpdateActions(ctx, tt.flowID, tt.actions).Return(tt.response, nil)
			res, err := h.FlowUpdateActions(ctx, tt.agent, tt.flowID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
