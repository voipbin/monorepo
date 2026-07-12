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
	cmresolution "monorepo/bin-contact-manager/models/resolution"

	"monorepo/bin-api-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

// Test_CaseResolutionCreate_DerivesResolvedByFromCaller verifies
// CaseResolutionCreate passes the caller's own a.AgentID() (not a
// client-suppliable value) as resolved_by_id, matching CaseClose's
// established pattern.
func Test_CaseResolutionCreate_DerivesResolvedByFromCaller(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	resolutionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

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
		Return(&cmkase.Case{ID: caseID, CustomerID: customerID, Status: cmkase.StatusOpen}, nil)

	// The key assertion: ContactV1CaseResolutionCreate must be called
	// with the caller's OWN agentID (a.AgentID()) as resolved_by_id --
	// gomock's exact-match EXPECT() below fails the test if the code
	// ever accepts/forwards a different, client-suppliable agent ID.
	mockReq.EXPECT().
		ContactV1CaseResolutionCreate(ctx, customerID, caseID, contactID, "positive", string(commonidentity.OwnerTypeAgent), agentID).
		Return(&cmresolution.Resolution{ID: resolutionID, CaseID: &caseID, ContactID: contactID}, nil)

	res, err := h.CaseResolutionCreate(ctx, a, caseID, contactID, "positive")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if res == nil {
		t.Errorf("Expected result but got nil")
	}
}

// Test_CaseResolutionCreate_CrossTenantCase verifies a cross-tenant case
// ID is rejected (via caseGet's tenant check) before any resolution RPC
// is issued.
func Test_CaseResolutionCreate_CrossTenantCase(t *testing.T) {
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

	// No ContactV1CaseResolutionCreate EXPECT() set -- gomock fails the
	// test if the resolution RPC is ever reached for a case that failed
	// the tenant check.
	_, err := h.CaseResolutionCreate(ctx, a, caseID, contactID, "positive")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseResolutionCreate_PermissionDenied verifies a non-admin/manager
// agent is rejected.
func Test_CaseResolutionCreate_PermissionDenied(t *testing.T) {
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
		Permission: amagent.PermissionNone,
	})

	mockReq.EXPECT().
		ContactV1CaseGet(ctx, customerID, caseID).
		Return(&cmkase.Case{ID: caseID, CustomerID: customerID, Status: cmkase.StatusOpen}, nil)

	_, err := h.CaseResolutionCreate(ctx, a, caseID, contactID, "positive")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseResolutionCreate_DirectAccessDenied verifies direct-access
// (accesskey) identities are rejected.
func Test_CaseResolutionCreate_DirectAccessDenied(t *testing.T) {
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

	_, err := h.CaseResolutionCreate(ctx, a, caseID, contactID, "positive")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

// Test_CaseResolutionDelete_DerivesTenantFromCaller verifies
// CaseResolutionDelete performs the tenant check via caseGet before
// delegating to ContactV1CaseResolutionDelete.
func Test_CaseResolutionDelete_DerivesTenantFromCaller(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	resolutionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

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
		Return(&cmkase.Case{ID: caseID, CustomerID: customerID, Status: cmkase.StatusOpen}, nil)

	mockReq.EXPECT().
		ContactV1CaseResolutionDelete(ctx, customerID, caseID, resolutionID).
		Return(nil)

	if err := h.CaseResolutionDelete(ctx, a, caseID, resolutionID); err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

// Test_CaseResolutionDelete_CrossTenantCase verifies a cross-tenant case
// ID is rejected before any delete RPC is issued.
func Test_CaseResolutionDelete_CrossTenantCase(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	resolutionID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

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

	if err := h.CaseResolutionDelete(ctx, a, caseID, resolutionID); err == nil {
		t.Errorf("Expected error but got none")
	}
}
