package servicehandler

import (
	"context"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cmkase "monorepo/bin-contact-manager/models/kase"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_CaseList(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("6f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		size   uint64
		token  string
		status string

		expectCustomerID uuid.UUID
		responseItems    []*cmkase.Case
		responseToken    string
		expectErr        bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  20,
			token: "",

			expectCustomerID: customerID,
			responseItems:    []*cmkase.Case{},
			responseToken:    "",
			expectErr:        false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			size:  20,
			token: "",

			expectErr: true,
		},
		{
			name: "superadmin can list cases across customers",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: otherCustomerID,
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			}),
			size:  20,
			token: "",

			// The superadmin caller's own CustomerID is otherCustomerID, but
			// CaseList is called with customerID (the target customer being
			// investigated) -- proving the cross-customer bypass is inherited
			// automatically via hasPermission, the same way every other
			// resource in this package gets it (etc.go), with no case-specific
			// authorization code added.
			expectCustomerID: customerID,
			responseItems:    []*cmkase.Case{},
			responseToken:    "",
			expectErr:        false,
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

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1CaseList(ctx, tt.expectCustomerID, tt.status, "", uuid.Nil, tt.size, tt.token).
					Return(tt.responseItems, tt.responseToken, nil)
			}

			items, _, err := h.CaseList(ctx, tt.agent, tt.expectCustomerID, tt.size, tt.token, tt.status, "", uuid.Nil)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if items == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_CaseListUnresolved(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		size  uint64
		token string

		responseItems []*cmkase.Case
		expectErr     bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			size:  20,
			token: "",

			responseItems: []*cmkase.Case{},
			expectErr:     false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionNone,
			}),
			size:  20,
			token: "",

			expectErr: true,
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

			if !tt.expectErr {
				mockReq.EXPECT().
					ContactV1CaseListUnresolved(ctx, tt.agent.CustomerID, tt.size, tt.token).
					Return(tt.responseItems, "", nil)
			}

			items, _, err := h.CaseListUnresolved(ctx, tt.agent, tt.size, tt.token)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if items == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func Test_CaseGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		responseCase *cmkase.Case
		expectErr    bool
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: false,
		},
		{
			name: "permission denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         agentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCase: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
			expectErr: true,
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

			mockReq.EXPECT().
				ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).
				Return(tt.responseCase, nil)

			res, err := h.CaseGet(ctx, tt.agent, tt.caseID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if res == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

// Test_CaseGet_NotFoundSentinel exercises the not-found passthrough path so
// error mapping is exercised, not just success.
func Test_CaseGet_ErrPassthrough(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(nil, serviceerrors.ErrNotFound)

	_, err := h.CaseGet(ctx, a, caseID)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseClose_DerivesClosedByFromCaller is a regression test (round-1
// Phase 5 review defect): closed_by_id must be derived server-side from
// the caller's own agent identity (a.AgentID()), never accepted as
// client input -- otherwise the closing-agent attribution the platform
// treats as a hard invariant (design §5.3) could be forged by any agent
// with close permission, including attributing a closure to an agent in
// a different tenant.
func Test_CaseClose_DerivesClosedByFromCaller(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
		}, nil)

	// The key assertion: ContactV1CaseClose must be called with the
	// caller's OWN agentID (a.AgentID()) -- gomock's exact-match
	// EXPECT() below fails the test if the code ever accepts/forwards a
	// different, client-suppliable agent ID.
	mockReq.EXPECT().
		ContactV1CaseClose(ctx, customerID, caseID, string(commonidentity.OwnerTypeAgent), agentID).
		Return(&cmkase.Case{ID: caseID, CustomerID: customerID, Status: cmkase.StatusClosed}, nil)

	res, err := h.CaseClose(ctx, a, caseID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res == nil {
		t.Errorf("Expected result but got nil")
	}
}

// Test_CaseClose_CrossTenant verifies a cross-tenant case ID is rejected
// (via caseGet's tenant check) before any close RPC is issued.
func Test_CaseClose_CrossTenant(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(nil, serviceerrors.ErrNotFound)

	// No ContactV1CaseClose EXPECT() set -- gomock fails the test if the
	// close RPC is ever reached for a case that failed the tenant check.
	_, err := h.CaseClose(ctx, a, caseID)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseContinue_UsesCallerAgentID verifies CaseContinue passes the
// caller's own a.AgentID() (not a client-suppliable value) as the acting
// agent, and derives callerIsAdmin from the caller's own permission
// bitmask.
func Test_CaseContinue_UsesCallerAgentID(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	newCaseID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusClosed,
		}, nil)

	mockReq.EXPECT().
		ContactV1CaseContinue(ctx, customerID, caseID, string(commonidentity.OwnerTypeAgent), agentID, true).
		Return(&cmkase.Case{ID: newCaseID, CustomerID: customerID, Status: cmkase.StatusOpen}, nil)

	res, err := h.CaseContinue(ctx, a, caseID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res == nil {
		t.Errorf("Expected result but got nil")
	}
}
