package servicehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	fmflow "monorepo/bin-flow-manager/models/flow"

	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

func Test_WebchatWidgetGet(t *testing.T) {
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

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		Name:     "Support Widget",
		Status:   wcwidget.StatusActive,
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)

	res, err := h.WebchatWidgetGet(ctx, a, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := responseWidget.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

func Test_WebchatWidgetGet_PermissionDenied(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	widgetOwnerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	requesterID := uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	// requester belongs to a DIFFERENT customer than the widget owner --
	// must be rejected regardless of the requester's own permission bits.
	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: requesterID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: widgetOwnerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)

	if _, err := h.WebchatWidgetGet(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

func Test_WebchatWidgetGet_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
	ctx := context.Background()

	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	// A visitor's direct-scope JWT must never be able to read Widget
	// admin config (theme, flow_id, welcome_message) -- only
	// SessionCreate/MessageCreate/SessionEnd are direct-reachable.
	if _, err := h.WebchatWidgetGet(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: direct access not supported error, got: ok")
	}
}

// Test_WebchatWidgetCreate verifies the happy path: an admin creating a
// widget with a flow_id that genuinely belongs to their own customer.
func Test_WebchatWidgetCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	flowID := uuid.FromStringOrNil("cc847807-6cc4-4713-9dec-53a42840e74c")
	widgetID := uuid.FromStringOrNil("dd7defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	flow := &fmflow.Flow{
		Identity: commonidentity.Identity{ID: flowID, CustomerID: customerID},
	}
	responseWidget := &wcwidget.Widget{
		Identity:      commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		SessionFlowID: flowID,
	}

	mockReq.EXPECT().FlowV1FlowGet(ctx, flowID).Return(flow, nil)
	mockReq.EXPECT().WebchatV1WidgetCreate(ctx, customerID, "widget", "welcome", flowID, uuid.Nil, 300, gomock.Any()).Return(responseWidget, nil)

	res, err := h.WebchatWidgetCreate(ctx, a, "widget", "welcome", flowID, uuid.Nil, 300, nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := responseWidget.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatWidgetCreate_CrossTenant_WrongFlowOwner is a regression test
// for a cross-tenant flow-execution vector: an admin must not be able to
// point their own customer's widget at a flow_id belonging to a DIFFERENT
// customer. Without this check, the first inbound message on that
// widget's session would trigger the other customer's Flow using the
// caller's own customer_id as the activeflow owner -- a cross-tenant
// flow-execution/data-leak bug, found via round 4 of an independent
// adversarial code review.
func Test_WebchatWidgetCreate_CrossTenant_WrongFlowOwner(t *testing.T) {
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
	victimFlowID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	victimFlow := &fmflow.Flow{
		Identity: commonidentity.Identity{ID: victimFlowID, CustomerID: victimCustomerID},
	}

	mockReq.EXPECT().FlowV1FlowGet(ctx, victimFlowID).Return(victimFlow, nil)

	if _, err := h.WebchatWidgetCreate(ctx, a, "widget", "welcome", victimFlowID, uuid.Nil, 300, nil); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatWidgetUpdate_CrossTenant_WrongFlowOwner is the Update-path
// equivalent of the above: an admin must not be able to repoint an
// EXISTING widget (that they do own) at a flow_id belonging to a
// different customer.
func Test_WebchatWidgetUpdate_CrossTenant_WrongFlowOwner(t *testing.T) {
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
	widgetID := uuid.FromStringOrNil("ee7defde-ad5e-11ed-a8c3-7bc19647b03f")
	victimFlowID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	ownWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: callerCustomerID},
	}
	victimFlow := &fmflow.Flow{
		Identity: commonidentity.Identity{ID: victimFlowID, CustomerID: victimCustomerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(ownWidget, nil)
	mockReq.EXPECT().FlowV1FlowGet(ctx, victimFlowID).Return(victimFlow, nil)

	if _, err := h.WebchatWidgetUpdate(ctx, a, widgetID, "widget", "welcome", victimFlowID, uuid.Nil, 300, nil); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatWidgetUpdate_SuperAdmin_CrossTenant_WrongFlowOwner is a
// regression test for round 5 of an independent adversarial code review:
// a ProjectSuperAdmin caller (whose own a.CustomerID is DIFFERENT from
// both the target widget's owner and the flow's owner) must not be able
// to repoint Customer B's widget at a flow that happens to belong to the
// superadmin's OWN tenant. The buggy code compared f.CustomerID against
// the caller's a.CustomerID (a tautology satisfied here) instead of the
// widget's actual owner (w.CustomerID), silently reintroducing the
// cross-tenant flow-execution vector the round 4 fix was meant to close,
// gated behind the superadmin privilege level.
func Test_WebchatWidgetUpdate_SuperAdmin_CrossTenant_WrongFlowOwner(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	superAdminCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	widgetID := uuid.FromStringOrNil("ee7defde-ad5e-11ed-a8c3-7bc19647b03f")
	// The flow belongs to the SUPERADMIN's own tenant, not the widget's.
	sameTenantAsCallerFlowID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: superAdminCustomerID,
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})

	victimWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: victimCustomerID},
	}
	callerTenantFlow := &fmflow.Flow{
		Identity: commonidentity.Identity{ID: sameTenantAsCallerFlowID, CustomerID: superAdminCustomerID},
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(victimWidget, nil)
	mockReq.EXPECT().FlowV1FlowGet(ctx, sameTenantAsCallerFlowID).Return(callerTenantFlow, nil)

	if _, err := h.WebchatWidgetUpdate(ctx, a, widgetID, "widget", "welcome", sameTenantAsCallerFlowID, uuid.Nil, 300, nil); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatWidgetGet_SoftDeleted is a regression test for round 7's
// finding: widgetGet must reject a soft-deleted widget (TMDelete set),
// mirroring flowGet's established "deleted resources are not found"
// pattern. Without this, WebchatWidgetGet could return full config for a
// deleted widget, WebchatWidgetUpdate could silently resurrect one, and
// WebchatWidgetDirectHashRegenerate could mint fresh visitor-facing
// credentials for a widget that should be gone.
func Test_WebchatWidgetGet_SoftDeleted(t *testing.T) {
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
	deletedAt := time.Now()

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	deletedWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		TMDelete: &deletedAt,
	}

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(deletedWidget, nil)

	if _, err := h.WebchatWidgetGet(ctx, a, widgetID); err == nil {
		t.Error("Wrong match. expect: not found error, got: ok")
	}
}
