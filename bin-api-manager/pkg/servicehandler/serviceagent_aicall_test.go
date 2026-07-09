package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

// Test_ServiceAgentAIcallList verifies a plain Agent permission (not
// Admin/Manager) can list its own customer's aicalls via the service_agents
// surface, optionally scoped by reference_type/reference_id.
func Test_ServiceAgentAIcallList(t *testing.T) {

	tests := []struct {
		name string

		agent         *auth.AuthIdentity
		pageToken     string
		pageSize      uint64
		referenceType string
		referenceID   uuid.UUID
		status        string

		response []amaicall.AIcall

		expectFilters map[amaicall.Field]any
		expectRes     []*amaicall.WebhookMessage
	}{
		{
			"agent permission, no reference filter",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"",
			uuid.Nil,
			"",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:    false,
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},
		},
		{
			"agent permission, filtered by reference_type and reference_id",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"contact_case",
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			"",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID:    uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:       false,
				amaicall.FieldReferenceType: "contact_case",
				amaicall.FieldReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},
		},
		{
			"agent permission, filtered by reference_type, reference_id, and status",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			"2020-10-20T01:00:00.995000Z",
			10,
			"contact_case",
			uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			"progressing",

			[]amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
				},
			},

			map[amaicall.Field]any{
				amaicall.FieldCustomerID:    uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				amaicall.FieldDeleted:       false,
				amaicall.FieldReferenceType: "contact_case",
				amaicall.FieldReferenceID:   uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				amaicall.FieldStatus:        "progressing",
			},
			[]*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
					},
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)
			res, err := h.ServiceAgentAIcallList(ctx, tt.agent, tt.pageSize, tt.pageToken, tt.referenceType, tt.referenceID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAIcallCreate verifies a plain Agent permission (not
// Admin/Manager) can create an aicall belonging to its own customer via the
// service_agents surface, and that an activeflow is created automatically.
func Test_ServiceAgentAIcallCreate(t *testing.T) {

	type test struct {
		name string

		agent          *auth.AuthIdentity
		assistanceType amaicall.AssistanceType
		assistanceID   uuid.UUID
		referenceType  amaicall.ReferenceType
		referenceID    uuid.UUID

		responseAI         *amai.AI
		responseActiveflow *fmactiveflow.Activeflow
		responseAIcall     *amaicall.AIcall

		expectRes *amaicall.WebhookMessage
	}

	tests := []test{
		{
			name: "agent permission, ai assistance",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			referenceType:  amaicall.ReferenceTypeContactCase,
			referenceID:    uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000010"),
				},
			},
			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
			},

			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			switch tt.assistanceType {
			case amaicall.AssistanceTypeAI:
				mockReq.EXPECT().AIV1AIGet(ctx, tt.assistanceID).Return(tt.responseAI, nil)
			case amaicall.AssistanceTypeTeam:
				mockReq.EXPECT().AIV1TeamGet(ctx, tt.assistanceID).Return(nil, nil)
			}

			mockReq.EXPECT().FlowV1ActiveflowCreate(
				ctx,
				uuid.Nil,
				tt.agent.CustomerID,
				uuid.Nil,
				fmactiveflow.ReferenceTypeAPI,
				uuid.Nil,
				uuid.Nil,
				gomock.Any(),
				gomock.Any(),
				gomock.Any(),
			).Return(tt.responseActiveflow, nil)

			mockReq.EXPECT().AIV1AIcallStart(
				ctx,
				tt.assistanceType,
				tt.assistanceID,
				tt.responseActiveflow.ID,
				tt.referenceType,
				tt.referenceID,
			).Return(tt.responseAIcall, nil)

			res, err := h.ServiceAgentAIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_ServiceAgentAIcallCreate_TenantIsolation verifies that when the
// resolved assistance entity's customer does not match the calling agent's
// own customer, the request is rejected -- tenant isolation without any
// ownership/role check beyond that.
func Test_ServiceAgentAIcallCreate_TenantIsolation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &serviceHandler{
		reqHandler:  mockReq,
		dbHandler:   mockDB,
		utilHandler: mockUtil,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAgent,
	})
	assistanceID := uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46")

	mockReq.EXPECT().AIV1AIGet(ctx, assistanceID).Return(&amai.AI{
		Identity: commonidentity.Identity{
			ID:         assistanceID,
			CustomerID: uuid.FromStringOrNil("other-customer-does-not-match"),
		},
	}, nil)

	_, err := h.ServiceAgentAIcallCreate(ctx, agent, amaicall.AssistanceTypeAI, assistanceID, amaicall.ReferenceTypeContactCase, uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"))
	if err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}
