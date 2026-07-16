package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	wcsession "monorepo/bin-webchat-manager/models/session"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

// Test_WebchatSessionCreate_Direct verifies the visitor-facing path: a
// direct-scope JWT issued by POST /auth/boot to a specific Widget can
// create a Session for that same widget. This is the v7 core flow
// (design doc §7/§14): session creation decoupled from first message,
// authenticated purely by the widget's direct hash, no agent/customer
// JWT involved.
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
