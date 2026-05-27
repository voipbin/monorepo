package servicehandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	amteam "monorepo/bin-ai-manager/models/team"
	dmdirect "monorepo/bin-direct-manager/models/direct"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_AIcallCreate(t *testing.T) {

	type test struct {
		name string

		agent          *auth.AuthIdentity
		assistanceType amaicall.AssistanceType
		assistanceID   uuid.UUID
		referenceType  amaicall.ReferenceType
		referenceID    uuid.UUID

		responseAI         *amai.AI
		responseTeam       *amteam.Team
		responseActiveflow *fmactiveflow.Activeflow
		responseAIcall     *amaicall.AIcall

		// expectAssistanceType is the type passed to AIV1AIcallStart after normalization
		expectAssistanceType amaicall.AssistanceType
		expectRes            *amaicall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			assistanceType: amaicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			referenceType:  amaicall.ReferenceTypeCall,
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

			expectAssistanceType: amaicall.AssistanceTypeAI,
			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
			},
		},
		{
			name: "ai_team normalized to team",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			assistanceType: amaicall.AssistanceType(dmdirect.ResourceTypeAITeam),
			assistanceID:   uuid.FromStringOrNil("b1c2d3e4-0000-0000-0000-000000000001"),
			referenceType:  amaicall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseTeam: &amteam.Team{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1c2d3e4-0000-0000-0000-000000000001"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseActiveflow: &fmactiveflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-0000-0000-0000-000000000020"),
				},
			},
			responseAIcall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2d3e4f5-0000-0000-0000-000000000001"),
				},
			},

			expectAssistanceType: amaicall.AssistanceTypeTeam,
			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2d3e4f5-0000-0000-0000-000000000001"),
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

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			// set up mocks based on the expected normalized type
			switch tt.expectAssistanceType {
			case amaicall.AssistanceTypeAI:
				mockReq.EXPECT().AIV1AIGet(ctx, tt.assistanceID).Return(tt.responseAI, nil)
			case amaicall.AssistanceTypeTeam:
				mockReq.EXPECT().AIV1TeamGet(ctx, tt.assistanceID).Return(tt.responseTeam, nil)
			}

			mockReq.EXPECT().FlowV1ActiveflowCreate(
				ctx,
				uuid.Nil,
				tt.agent.CustomerID,
				uuid.Nil,
				fmactiveflow.ReferenceTypeAPI,
				uuid.Nil,
				uuid.Nil,
			).Return(tt.responseActiveflow, nil)

			mockReq.EXPECT().AIV1AIcallStart(
				ctx,
				tt.expectAssistanceType,
				tt.assistanceID,
				tt.responseActiveflow.ID,
				tt.referenceType,
				tt.referenceID,
			).Return(tt.responseAIcall, nil)

			res, err := h.AIcallCreate(ctx, tt.agent, tt.assistanceType, tt.assistanceID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallListByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *auth.AuthIdentity
		size    uint64
		token   string
		filters map[amaicall.Field]any

		response  []amaicall.AIcall
		expectRes []*amaicall.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  10,
			token: "2020-09-20T03:23:20.995000Z",
			filters: map[amaicall.Field]any{
				amaicall.FieldDeleted:    false,
				amaicall.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			response: []amaicall.AIcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
					},
				},
			},
			expectRes: []*amaicall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallList(ctx, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.AIcallGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallGet(t *testing.T) {

	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aicallID uuid.UUID

		response  *amaicall.AIcall
		expectRes *amaicall.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),

			response: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.response, nil)

			res, err := h.AIcallGet(ctx, tt.agent, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallDelete(t *testing.T) {

	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aicallID uuid.UUID

		responseAicall *amaicall.AIcall
		expectRes      *amaicall.WebhookMessage
	}{
		{
			name: "normal",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAicall, nil)
			mockReq.EXPECT().AIV1AIcallDelete(ctx, tt.aicallID).Return(tt.responseAicall, nil)

			res, err := h.AIcallDelete(ctx, tt.agent, tt.aicallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIcallTerminate(t *testing.T) {

	tests := []struct {
		name string

		agent    *auth.AuthIdentity
		aicallID uuid.UUID

		responseAicall    *amaicall.AIcall
		responseAicallErr error

		expectTerminateCall  bool
		responseTerminate    *amaicall.AIcall
		responseTerminateErr error

		wantErr   bool
		expectRes *amaicall.WebhookMessage
	}{
		{
			name: "normal - agent with CustomerAdmin",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectTerminateCall: true,
			responseTerminate: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			wantErr: false,
			expectRes: &amaicall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
		{
			name: "aicallGet failure",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicallErr: fmt.Errorf("not found"),

			wantErr: true,
		},
		{
			name: "agent permission denied",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			wantErr: true,
		},
		{
			name: "direct token - resource type not allowed",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				AllowedResourceTypes: []string{"call"},
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			wantErr: true,
		},
		{
			name: "direct token - customer ID mismatch",

			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID:           uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000000"),
				AllowedResourceTypes: []string{"aicall"},
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			wantErr: true,
		},
		{
			name: "RPC failure",

			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			aicallID: uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			responseAicall: &amaicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},

			expectTerminateCall:  true,
			responseTerminateErr: fmt.Errorf("rpc error"),

			wantErr: true,
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

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAicall, tt.responseAicallErr)
			if tt.expectTerminateCall {
				mockReq.EXPECT().AIV1AIcallTerminate(ctx, tt.aicallID).Return(tt.responseTerminate, tt.responseTerminateErr)
			}

			res, err := h.AIcallTerminate(ctx, tt.agent, tt.aicallID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
