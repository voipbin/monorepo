package servicehandler

import (
	"context"
	"reflect"
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
					ContactV1CaseList(ctx, tt.expectCustomerID, tt.status, "", uuid.Nil, uuid.Nil, tt.size, tt.token, "").
					Return(tt.responseItems, tt.responseToken, nil)
			}

			items, _, err := h.CaseList(ctx, tt.agent, tt.expectCustomerID, tt.size, tt.token, tt.status, "", uuid.Nil, uuid.Nil, "")
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

// Test_CaseUpdateContact_DirectAccessRejected verifies direct-access
// callers (a.IsDirect() == true) are rejected with
// ErrDirectAccessNotSupported before any RPC is issued (design
// VOIP-1253).
func Test_CaseUpdateContact_DirectAccessRejected(t *testing.T) {
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}

	ctx := context.Background()
	a := auth.NewDirectIdentity(&auth.DirectScope{CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")})

	// No ContactV1CaseGet/ContactV1CaseUpdateContact EXPECT() set --
	// gomock fails the test if either RPC is reached for a direct-access
	// caller.
	_, err := h.CaseUpdateContact(ctx, a, caseID, contactID)
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("Expected ErrDirectAccessNotSupported, got: %v", err)
	}
}

// Test_CaseUpdateContact_PermissionDenied verifies a caller lacking
// PermissionCustomerAdmin/PermissionCustomerManager is rejected with
// ErrPermissionDenied after the case is found but before the update RPC
// is issued.
func Test_CaseUpdateContact_PermissionDenied(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

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
		Permission: amagent.PermissionCustomerAgent,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{
			ID:         caseID,
			CustomerID: customerID,
			Status:     cmkase.StatusOpen,
		}, nil)

	// No ContactV1CaseUpdateContact EXPECT() set -- gomock fails the
	// test if the update RPC is ever reached for a caller lacking the
	// required permission.
	_, err := h.CaseUpdateContact(ctx, a, caseID, contactID)
	if err != serviceerrors.ErrPermissionDenied {
		t.Errorf("Expected ErrPermissionDenied, got: %v", err)
	}
}

// Test_CaseUpdateContact_CrossTenant verifies a cross-tenant case ID is
// rejected (via caseGet's tenant check) before any update RPC is
// issued.
func Test_CaseUpdateContact_CrossTenant(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

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

	// No ContactV1CaseUpdateContact EXPECT() set -- gomock fails the
	// test if the update RPC is ever reached for a case that failed the
	// tenant check.
	_, err := h.CaseUpdateContact(ctx, a, caseID, contactID)
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseUpdateContact_Success verifies a permitted, non-direct caller
// delegates to ContactV1CaseUpdateContact with the exact
// (customerID, id, contactID) arguments and returns its result. This is
// exercised with a non-Nil contactID (the "attach" path); the
// contact_id="" -> uuid.Nil ("detach") conversion itself is an HTTP-layer
// concern covered in server/contact_cases_test.go
// (Test_PutContactCasesId), since CaseUpdateContact only ever forwards
// whatever uuid.UUID it is handed.
func Test_CaseUpdateContact_Success(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")

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

	expectRes := &cmkase.Case{
		ID:         caseID,
		CustomerID: customerID,
		ContactID:  &contactID,
	}
	mockReq.EXPECT().
		ContactV1CaseUpdateContact(ctx, customerID, caseID, contactID).
		Return(expectRes, nil)

	res, err := h.CaseUpdateContact(ctx, a, caseID, contactID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res != expectRes {
		t.Errorf("Expected result %v, got: %v", expectRes, res)
	}
}

// Test_CaseUpdateContact_DetachToNil verifies CaseUpdateContact forwards
// a uuid.Nil contactID as-is to ContactV1CaseUpdateContact (the
// "detach" path) -- servicehandler performs no conversion of its own,
// it merely passes through whatever contactID the HTTP layer already
// resolved (see server/contact_cases_test.go for the actual
// contact_id="" -> uuid.Nil boundary conversion coverage).
func Test_CaseUpdateContact_DetachToNil(t *testing.T) {
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

	expectRes := &cmkase.Case{
		ID:         caseID,
		CustomerID: customerID,
		ContactID:  nil,
	}
	// The key assertion: contactID is forwarded as exactly uuid.Nil,
	// not some other zero-ish sentinel -- gomock's exact-match EXPECT()
	// fails the test otherwise.
	mockReq.EXPECT().
		ContactV1CaseUpdateContact(ctx, customerID, caseID, uuid.Nil).
		Return(expectRes, nil)

	res, err := h.CaseUpdateContact(ctx, a, caseID, uuid.Nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res != expectRes {
		t.Errorf("Expected result %v, got: %v", expectRes, res)
	}
}

// Test_CaseAssign covers the Admin/Manager top-level case owner assign
// endpoint (VOIP-1514). Mirrors Test_ServiceAgentCaseAssign's table
// structure but with Admin/Manager/plain-agent/direct-access identities
// instead of a single agent-permission identity, plus the cross-tenant
// case scenario.
func Test_CaseAssign(t *testing.T) {
	type test struct {
		name string

		agent   *auth.AuthIdentity
		caseID  uuid.UUID
		ownerID uuid.UUID

		expectCaseGetErr error
		responseCaseGet  *cmkase.Case

		expectAgentGetCall bool
		responseAgentGet   *amagent.Agent
		responseAgentErr   error

		expectAssignCall   bool
		responseCaseAssign *cmkase.Case

		expectErr   bool
		expectErrIs error
	}

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	otherCustomerID := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	adminID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb")
	ownerID := uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6")

	tests := []test{
		{
			name: "admin permission, valid same-customer owner",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectAgentGetCall: true,
			responseAgentGet: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerID,
					CustomerID: customerID,
				},
			},

			expectAssignCall: true,
			responseCaseAssign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerID,
				},
			},
		},
		{
			name: "manager permission, valid same-customer owner",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectAgentGetCall: true,
			responseAgentGet: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerID,
					CustomerID: customerID,
				},
			},

			expectAssignCall: true,
			responseCaseAssign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerID,
				},
			},
		},
		{
			// A plain agent (not Admin/Manager) must be rejected on the
			// top-level endpoint -- ContactV1CaseGet is still called
			// first (caseGet precedes hasPermission in this group), but
			// neither AgentV1AgentGet nor ContactV1CaseAssign may fire.
			name: "plain agent permission is denied",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectAgentGetCall: false,
			expectAssignCall:   false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// Case is closed -- ErrCaseClosed, no owner lookup, no
			// assign RPC.
			name: "case is closed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusClosed,
			},

			expectAgentGetCall: false,
			expectAssignCall:   false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrCaseClosed,
		},
		{
			// Owner agent belongs to a different customer -- collapses
			// to ErrNotFound (anti-enumeration), matching
			// ServiceAgentCaseAssign's behavior.
			name: "owner agent belongs to a different customer",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectAgentGetCall: true,
			responseAgentGet: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerID,
					CustomerID: otherCustomerID,
				},
			},

			expectAssignCall: false,
			expectErr:        true,
			expectErrIs:      serviceerrors.ErrNotFound,
		},
		{
			// Case lookup itself fails (case does not exist) -- the
			// error propagates as-is; no permission check, no owner
			// lookup, no assign RPC.
			name: "case does not exist",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			expectCaseGetErr: serviceerrors.ErrNotFound,

			expectAgentGetCall: false,
			expectAssignCall:   false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrNotFound,
		},
		{
			// Cross-tenant case: the caller's own a.CustomerID is used
			// for ContactV1CaseGet (caseGet), which already
			// tenant-filters on the contact-manager side -- a
			// cross-tenant case ID resolves to ErrNotFound at the
			// caseGet step itself, never reaching hasPermission (design
			// §9, round-3 review).
			name: "cross-tenant case resolves to not found at caseGet",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID:  caseID,
			ownerID: ownerID,

			expectCaseGetErr: serviceerrors.ErrNotFound,

			expectAgentGetCall: false,
			expectAssignCall:   false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrNotFound,
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

			if tt.expectCaseGetErr != nil {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(nil, tt.expectCaseGetErr)
			} else {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)
			}

			if tt.expectAgentGetCall {
				mockReq.EXPECT().AgentV1AgentGet(ctx, tt.ownerID).Return(tt.responseAgentGet, tt.responseAgentErr)
			}
			if tt.expectAssignCall {
				mockReq.EXPECT().ContactV1CaseAssign(ctx, tt.agent.CustomerID, tt.caseID, tt.ownerID).Return(tt.responseCaseAssign, nil)
			}

			res, err := h.CaseAssign(ctx, tt.agent, tt.caseID, tt.ownerID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.expectErrIs != nil && err != tt.expectErrIs {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErrIs, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if reflect.DeepEqual(res, tt.responseCaseAssign) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseCaseAssign, res)
			}
		})
	}
}

// Test_CaseUnassign covers the design VOIP-1515 §8.1 table for the
// top-level CaseUnassign, including the ErrCaseClosed guard (item 9) and
// the caseGet -> permission -> closed-case check ordering regression
// (item 10: a non-admin, non-owning caller on a closed case must get
// ErrPermissionDenied, not ErrCaseClosed).
func Test_CaseUnassign(t *testing.T) {
	type test struct {
		name string

		agent  *auth.AuthIdentity
		caseID uuid.UUID

		expectCaseGetErr error
		responseCaseGet  *cmkase.Case

		expectUnassignCall   bool
		responseCaseUnassign *cmkase.Case

		expectErr   bool
		expectErrIs error
	}

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	adminID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	ownerAgentID := uuid.FromStringOrNil("f6b8b5f0-8270-11ed-9e5a-4bcaa2b972d6")
	otherAgentID := uuid.FromStringOrNil("aaaaaaaa-0000-0000-0000-000000000001")
	caseID := uuid.FromStringOrNil("df394b78-8270-11ed-914d-6bceafeffecb")

	tests := []test{
		{
			// 1. Admin unassigns a case owned by someone else.
			name: "admin unassigns a case owned by someone else",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: true,
			responseCaseUnassign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
		},
		{
			// 2. Manager unassigns a case owned by someone else.
			name: "manager unassigns a case owned by someone else",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerManager,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: true,
			responseCaseUnassign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
		},
		{
			// 3. Owning agent self-unassigns.
			name: "owning agent self-unassigns",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         ownerAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: true,
			responseCaseUnassign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
		},
		{
			// 4. Non-owning agent (no admin/manager permission) attempts
			// unassign -> ErrPermissionDenied, RPC never called.
			name: "non-owning agent attempts unassign",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 5. Agent with no permission at all attempts unassign on an
			// unowned case -> ErrPermissionDenied.
			name: "no-permission agent on an unowned case",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 8. Unassigning an already-unowned case as admin/manager is a
			// no-op success.
			name: "admin unassigns an already-unowned case (idempotent no-op)",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},

			expectUnassignCall: true,
			responseCaseUnassign: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusOpen,
			},
		},
		{
			// 9. Closed case as admin -> ErrCaseClosed, caseGet is called
			// but ContactV1CaseUnassign is never called.
			name: "closed case as admin returns ErrCaseClosed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusClosed,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrCaseClosed,
		},
		{
			// 10. Non-admin, non-owning agent attempts unassign on a
			// CLOSED case -> ErrPermissionDenied, NOT ErrCaseClosed. This
			// is the check-ordering regression test: permission is
			// evaluated before the closed-case guard.
			name: "non-admin non-owner on closed case returns ErrPermissionDenied not ErrCaseClosed",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         otherAgentID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAgent,
			}),
			caseID: caseID,

			responseCaseGet: &cmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Status:     cmkase.StatusClosed,
				Owner: commonidentity.Owner{
					OwnerType: commonidentity.OwnerTypeAgent,
					OwnerID:   ownerAgentID,
				},
			},

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrPermissionDenied,
		},
		{
			// 7. Cross-tenant / nonexistent case id -> caseGet returns
			// ErrNotFound (propagated), permission check never reached.
			name: "case does not exist propagates caseGet error",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         adminID,
					CustomerID: customerID,
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			caseID: caseID,

			expectCaseGetErr: serviceerrors.ErrNotFound,

			expectUnassignCall: false,
			expectErr:          true,
			expectErrIs:        serviceerrors.ErrNotFound,
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

			if tt.expectCaseGetErr != nil {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(nil, tt.expectCaseGetErr)
			} else {
				mockReq.EXPECT().ContactV1CaseGet(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseGet, nil)
			}

			if tt.expectUnassignCall {
				mockReq.EXPECT().ContactV1CaseUnassign(ctx, tt.agent.CustomerID, tt.caseID).Return(tt.responseCaseUnassign, nil)
			}

			res, err := h.CaseUnassign(ctx, tt.agent, tt.caseID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.expectErrIs != nil && err != tt.expectErrIs {
					t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErrIs, err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if reflect.DeepEqual(res, tt.responseCaseUnassign) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseCaseUnassign, res)
			}
		})
	}

	t.Run("direct access is not supported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)

		h := &serviceHandler{
			reqHandler: mockReq,
			dbHandler:  mockDB,
		}
		ctx := context.Background()

		a := auth.NewDirectIdentity(&auth.DirectScope{CustomerID: customerID})

		// No ContactV1CaseGet/ContactV1CaseUnassign EXPECT() set -- gomock
		// fails the test if any downstream call fires for a direct-access
		// caller.
		_, err := h.CaseUnassign(ctx, a, caseID)
		if err == nil {
			t.Errorf("Expected error but got none")
		}
		if err != serviceerrors.ErrDirectAccessNotSupported {
			t.Errorf("Wrong error. expect: %v, got: %v", serviceerrors.ErrDirectAccessNotSupported, err)
		}
	})
}
