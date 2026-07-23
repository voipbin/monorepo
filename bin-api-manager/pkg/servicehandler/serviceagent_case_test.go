package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmkase "monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentCaseList(t *testing.T) {

	tests := []struct {
		name string

		agent     *auth.AuthIdentity
		pageSize  uint64
		pageToken string

		responseCases     []*cmkase.Case
		responseNextToken string

		expectRes []*cmkase.Case
	}{
		{
			// Plain Agent permission (not Admin/Manager) must be able to list
			// its own customer's cases via the service_agents surface.
			name: "agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			pageSize:  10,
			pageToken: "2020-10-20T01:00:00.995000Z",

			responseCases: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				},
			},
			responseNextToken: "2020-10-21T01:00:00.995000Z",

			expectRes: []*cmkase.Case{
				{
					ID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
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

			mockReq.EXPECT().ContactV1CaseList(ctx, tt.agent.CustomerID, "", "", uuid.Nil, uuid.Nil, tt.pageSize, tt.pageToken, "").Return(tt.responseCases, tt.responseNextToken, nil)

			res, nextToken, err := h.ServiceAgentCaseList(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

			if nextToken != tt.responseNextToken {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseNextToken, nextToken)
			}
		})
	}
}

func Test_ServiceAgentCaseGet(t *testing.T) {

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		responseCase *cmkase.Case

		expectRes *cmkase.Case
	}{
		{
			name: "agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),

			responseCase: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},

			expectRes: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCase, nil)

			res, err := h.ServiceAgentCaseGet(ctx, tt.agent, tt.caseID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentCaseClose(t *testing.T) {

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		responseCaseGet   *cmkase.Case
		responseCaseClose *cmkase.Case

		expectRes *cmkase.Case
	}{
		{
			name: "agent permission",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),

			responseCaseGet: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			responseCaseClose: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Status:     cmkase.StatusClosed,
			},

			expectRes: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Status:     cmkase.StatusClosed,
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

			mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)
			mockReq.EXPECT().ContactV1CaseClose(ctx, tt.agent.CustomerID, tt.caseID, string(commonidentity.OwnerTypeAgent), tt.agent.AgentID()).Return(tt.responseCaseClose, nil)

			res, err := h.ServiceAgentCaseClose(ctx, tt.agent, tt.caseID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentCaseAssign(t *testing.T) {

	type test struct {
		name string

		agent   *auth.AuthIdentity
		caseID  uuid.UUID
		ownerID uuid.UUID

		responseCaseGet    *cmkase.Case
		responseAgentGet   *amagent.Agent
		responseAgentErr   error
		responseCaseAssign *cmkase.Case

		expectAgentGetCall bool
		expectAssignCall   bool
		expectRes          *cmkase.Case
		expectErr          bool
	}

	agentCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")

	tests := []test{
		{
			name: "agent permission, valid same-customer owner",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:  uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
			ownerID: uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),

			responseCaseGet: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: agentCustomerID,
			},
			responseAgentGet: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
					CustomerID: agentCustomerID,
				},
			},
			responseCaseAssign: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: agentCustomerID,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
				},
			},

			expectAgentGetCall: true,
			expectAssignCall:   true,
			expectRes: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: agentCustomerID,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
				},
			},
		},
		{
			// The owner agent lookup errors (agent doesn't exist). This must
			// collapse to ErrNotFound, not surface the raw lookup error.
			name: "owner agent does not exist",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:  uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
			ownerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),

			responseCaseGet: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: agentCustomerID,
			},
			responseAgentErr: serviceerrors.ErrNotFound,

			expectAgentGetCall: true,
			expectAssignCall:   false,
			expectErr:          true,
		},
		{
			// The owner agent exists but belongs to a DIFFERENT customer.
			// Must also collapse to ErrNotFound (anti-enumeration).
			name: "owner agent belongs to a different customer",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:  uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
			ownerID: uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),

			responseCaseGet: &cmkase.Case{
				ID:         uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb"),
				CustomerID: agentCustomerID,
			},
			responseAgentGet: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6"),
					CustomerID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
				},
			},

			expectAgentGetCall: true,
			expectAssignCall:   false,
			expectErr:          true,
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

			mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)

			if tt.expectAgentGetCall {
				mockReq.EXPECT().AgentV1AgentGet(ctx, tt.ownerID).Return(tt.responseAgentGet, tt.responseAgentErr)
			}
			if tt.expectAssignCall {
				mockReq.EXPECT().ContactV1CaseAssign(ctx, tt.agent.CustomerID, tt.caseID, tt.ownerID).Return(tt.responseCaseAssign, nil)
			}

			res, err := h.ServiceAgentCaseAssign(ctx, tt.agent, tt.caseID, tt.ownerID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
