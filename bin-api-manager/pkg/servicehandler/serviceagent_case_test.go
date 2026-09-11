package servicehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmkase "monorepo/bin-contact-manager/models/kase"

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
		expectErrIs        error
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
		{
			// The case is CLOSED. Must be rejected with ErrCaseClosed before
			// the owner agent is even looked up, and ContactV1CaseAssign
			// must never be called.
			name: "case is closed",
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
				Status:     cmkase.StatusClosed,
			},

			expectAgentGetCall: false,
			expectAssignCall:   false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrCaseClosed,
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
				if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
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

// Test_ServiceAgentCaseUnassign covers the design VOIP-1515 §8.2 table.
func Test_ServiceAgentCaseUnassign(t *testing.T) {
	type test struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		expectCaseGetCall  bool
		responseCaseGet    *cmkase.Case
		responseCaseGetErr error

		expectUnassignCall   bool
		responseCaseUnassign *cmkase.Case

		expectErr   bool
		expectErrIs error
	}

	agentCustomerID2 := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID2 := uuid.FromStringOrNil("6f621078-8e5f-11ee-97b2-cfe7337b701c")
	ownerAgentID2 := uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6")
	otherAgentID2 := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000002")
	caseID2 := uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb")

	tests := []test{
		{
			// 1. Owning agent unassigns own case -> success.
			name: "owning agent unassigns own case",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerAgentID2,
					CustomerID: agentCustomerID2,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID2,

			expectCaseGetCall: true,
			responseCaseGet: &cmkase.Case{
				ID:         caseID2,
				CustomerID: agentCustomerID2,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID2,
				},
			},

			expectUnassignCall: true,
			responseCaseUnassign: &cmkase.Case{
				ID:         caseID2,
				CustomerID: agentCustomerID2,
				Status:     cmkase.StatusOpen,
			},
		},
		{
			// 2. Agent attempts to unassign a case owned by a different
			// agent -> ErrPermissionDenied, RPC never called.
			name: "agent attempts to unassign a case owned by a different agent",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID2,
					CustomerID: agentCustomerID2,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID2,

			expectCaseGetCall: true,
			responseCaseGet: &cmkase.Case{
				ID:         caseID2,
				CustomerID: agentCustomerID2,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID2,
				},
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 3. Agent attempts to unassign an unowned case ->
			// ErrPermissionDenied.
			name: "agent attempts to unassign an unowned case",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID2,
					CustomerID: agentCustomerID2,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID2,

			expectCaseGetCall: true,
			responseCaseGet: &cmkase.Case{
				ID:         caseID2,
				CustomerID: agentCustomerID2,
				Status:     cmkase.StatusOpen,
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 4. Agent without PermissionAll on their own customer ->
			// ErrPermissionDenied before caseGet is even reached. Uses a
			// direct-scoped identity (HasPermission always false for
			// TypeDirect) to force the denial deterministically -- a
			// plain agent identity's HasPermission(PermissionAll) always
			// returns true regardless of the agent's own Permission
			// bitmask (Agent.HasPermission special-cases perm ==
			// PermissionAll), so denial can only be exercised via a
			// same-customer identity type that never satisfies it.
			name:   "agent without permission is denied before caseGet",
			agent:  auth.NewDirectIdentity(&auth.DirectScope{CustomerID: agentCustomerID2}),
			caseID: caseID2,

			expectCaseGetCall:  false,
			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 5. Cross-tenant case id -> caseGet returns ErrNotFound.
			name: "cross-tenant case id returns not found",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerAgentID2,
					CustomerID: otherCustomerID2,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID2,

			expectCaseGetCall:  true,
			responseCaseGetErr: serviceerrors.ErrNotFound,

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrNotFound,
		},
		{
			// 6. Closed case owned by the caller -> ErrCaseClosed, RPC
			// never called.
			name: "closed case owned by caller returns ErrCaseClosed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerAgentID2,
					CustomerID: agentCustomerID2,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID2,

			expectCaseGetCall: true,
			responseCaseGet: &cmkase.Case{
				ID:         caseID2,
				CustomerID: agentCustomerID2,
				Status:     cmkase.StatusClosed,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID2,
				},
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrCaseClosed,
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

			if tt.expectCaseGetCall {
				if tt.responseCaseGetErr != nil {
					mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(nil, tt.responseCaseGetErr)
				} else {
					mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)
				}
			}

			if tt.expectUnassignCall {
				mockReq.EXPECT().ContactV1CaseUnassign(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseUnassign, nil)
			}

			res, err := h.ServiceAgentCaseUnassign(ctx, tt.agent, tt.caseID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErrIs, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseCaseUnassign) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseCaseUnassign, res)
			}
		})
	}
}

func Test_ServiceAgentCaseUpdateContact(t *testing.T) {

	agentCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("6f621078-8e5f-11ee-97b2-cfe7337b701c")
	caseID := uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb")
	contactID := uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440001")

	type test struct {
		name string

		agent     *auth.AuthIdentity
		caseID    uuid.UUID
		contactID uuid.UUID

		responseCaseGet    *cmkase.Case
		responseCaseGetErr error
		responseUpdate     *cmkase.Case

		expectCaseGetCall bool
		expectUpdateCall  bool
		expectRes         *cmkase.Case
		expectErr         bool
	}

	tests := []test{
		{
			// Plain Agent permission (not Admin/Manager) must be able to
			// attach a contact via the service_agents surface.
			name: "agent permission, attach",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:    caseID,
			contactID: contactID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
			},
			responseUpdate: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
				ContactID:  &contactID,
			},

			expectCaseGetCall: true,
			expectUpdateCall:  true,
			expectRes: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
				ContactID:  &contactID,
			},
		},
		{
			// contactID == uuid.Nil detaches -- the handler passes it
			// through unchanged to ContactV1CaseUpdateContact.
			name: "agent permission, detach",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: agentCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:    caseID,
			contactID: uuid.Nil,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
				ContactID:  &contactID,
			},
			responseUpdate: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
			},

			expectCaseGetCall: true,
			expectUpdateCall:  true,
			expectRes: &cmkase.Case{
				ID:         caseID,
				CustomerID: agentCustomerID,
			},
		},
		{
			// A direct/accesskey-scoped identity is rejected before any
			// downstream call is made.
			name: "direct identity rejected",
			agent: auth.NewDirectIdentity(&auth.DirectScope{
				CustomerID: agentCustomerID,
			}),
			caseID:    caseID,
			contactID: contactID,

			expectCaseGetCall: false,
			expectUpdateCall:  false,
			expectErr:         true,
		},
		{
			// caseGet tenant-verifies -- a case belonging to a different
			// customer than the caller must not reach the update RPC.
			name: "case belongs to a different customer",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: otherCustomerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:    caseID,
			contactID: contactID,

			responseCaseGetErr: serviceerrors.ErrNotFound,

			expectCaseGetCall: true,
			expectUpdateCall:  false,
			expectErr:         true,
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

			if tt.expectCaseGetCall {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, tt.responseCaseGetErr)
			}
			if tt.expectUpdateCall {
				mockReq.EXPECT().ContactV1CaseUpdateContact(ctx, tt.agent.CustomerID, tt.caseID, tt.contactID).Return(tt.responseUpdate, nil)
			}

			res, err := h.ServiceAgentCaseUpdateContact(ctx, tt.agent, tt.caseID, tt.contactID)
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
