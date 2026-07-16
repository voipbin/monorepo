package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

// Test_WebchatSessionCreate_Direct verifies the visitor-facing path: a
// direct-scope JWT scoped to widget A can create a Session for widget A.
func Test_WebchatSessionCreate_Direct(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           customerID,
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	responseSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   wcsession.StatusActive,
	}
	widget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(widget, nil)
	mockReq.EXPECT().WebchatV1SessionCreate(ctx, customerID, widgetID).Return(responseSession, nil)

	res, err := h.WebchatSessionCreate(ctx, a, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := responseSession.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatSessionCreate_Direct_WrongWidgetScope verifies a direct-scope
// JWT issued for widget A cannot be reused to create a Session for
// widget B -- the DirectScope.ResourceID must match the requested
// widget_id exactly.
func Test_WebchatSessionCreate_Direct_WrongWidgetScope(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	scopedWidgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	requestedWidgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           customerID,
		ResourceType:         "webchat_widget",
		ResourceID:           scopedWidgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	if _, err := h.WebchatSessionCreate(ctx, a, requestedWidgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatSessionCreate_Direct_DisallowedResourceType verifies a
// direct-scope JWT without "webchat_session" in AllowedResourceTypes is
// rejected -- confirms the directResourceMapping wiring
// (webchat_widget -> webchat_session) is actually enforced here, not just
// documented.
func Test_WebchatSessionCreate_Direct_DisallowedResourceType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           customerID,
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{}, // empty -- mis-scoped token
	})

	if _, err := h.WebchatSessionCreate(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatSessionCreate_Agent_CrossTenant_WrongWidgetOwner is a
// regression test for a cross-tenant resource-linkage bug: an
// agent/accesskey with admin/manager permission on their OWN customer
// account must not be able to create a session for a widget_id belonging
// to a DIFFERENT customer -- doing so would trigger that other customer's
// configured Flow using the wrong customer_id as the activeflow owner.
// Before the fix, the a.IsAgent()||a.IsAccesskey() branch only checked
// hasPermission(ctx, a, a.CustomerID, ...) -- a tautology that never
// resolved the widget or verified it belonged to the caller.
func Test_WebchatSessionCreate_Agent_CrossTenant_WrongWidgetOwner(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	callerCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	victimWidgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	victimWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: victimWidgetID, CustomerID: victimCustomerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, victimWidgetID).Return(victimWidget, nil)

	if _, err := h.WebchatSessionCreate(ctx, a, victimWidgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatSessionCreate_Agent_SuperAdmin_CrossTenant_OwnerCustomerID is
// a regression test for a customer_id data-integrity bug: hasPermission
// short-circuits to true for PermissionProjectSuperAdmin regardless of
// a.CustomerID vs the widget's actual owner. Before this fix,
// WebchatSessionCreate unconditionally passed the caller's OWN a.CustomerID
// (the superadmin's, not the widget owner's) to WebchatV1SessionCreate --
// creating a session tagged with the WRONG customer_id, invisible to the
// widget owner's own WebchatSessionList/Get calls. Verified here by
// asserting the downstream RPC call is made with the WIDGET's customer_id,
// not the superadmin caller's.
func Test_WebchatSessionCreate_Agent_SuperAdmin_CrossTenant_OwnerCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	superAdminCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetOwnerCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	widgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: superAdminCustomerID,
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})

	widget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: widgetOwnerCustomerID},
	}
	responseSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: widgetOwnerCustomerID},
		WidgetID: widgetID,
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(widget, nil)
	// The mock strictly asserts widgetOwnerCustomerID -- if the handler
	// forwarded the superadmin's own a.CustomerID instead, gomock would
	// reject the call as unexpected and this test would fail.
	mockReq.EXPECT().WebchatV1SessionCreate(ctx, widgetOwnerCustomerID, widgetID).Return(responseSession, nil)

	if _, err := h.WebchatSessionCreate(ctx, a, widgetID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_WebchatSessionCreate_Direct_OwnerCustomerID_ReResolvedFromWidget is a
// regression test for defense-in-depth found in round 6: the direct-scope
// (visitor) branch must re-resolve ownerCustomerID from the widget's
// actual current owner (via widgetGet) rather than trusting the
// CustomerID claim baked into the direct-scope JWT at boot time --
// mirroring WebchatMessageCreate's direct branch, which does the same
// for ownerCustomerID = s.CustomerID. Verified by using a JWT whose
// DirectScope.CustomerID claim deliberately does NOT match the widget's
// real w.CustomerID (simulating a stale/rotated-ownership claim) and
// asserting the downstream RPC call uses the widget's real owner.
func Test_WebchatSessionCreate_Direct_OwnerCustomerID_ReResolvedFromWidget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	staleClaimCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	actualWidgetOwnerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           staleClaimCustomerID,
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	widget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: actualWidgetOwnerID},
	}
	responseSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: actualWidgetOwnerID},
		WidgetID: widgetID,
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(widget, nil)
	// The mock strictly asserts actualWidgetOwnerID -- if the handler
	// forwarded the JWT's stale staleClaimCustomerID instead, gomock
	// would reject the call as unexpected and this test would fail.
	mockReq.EXPECT().WebchatV1SessionCreate(ctx, actualWidgetOwnerID, widgetID).Return(responseSession, nil)

	if _, err := h.WebchatSessionCreate(ctx, a, widgetID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}
