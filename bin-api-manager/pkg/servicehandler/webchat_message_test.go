package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

// Test_WebchatMessageCreate_Direct verifies the visitor-facing path: a
// direct-scope JWT scoped to widget A can post a message into a session
// that genuinely belongs to widget A.
func Test_WebchatMessageCreate_Direct(t *testing.T) {
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
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9f5e-4762c3c1e4e5")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           customerID,
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	responseSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
	}
	responseMessage := &wcmessage.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: wcmessage.DirectionInbound,
		Text:      "hello",
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(responseSession, nil)
	mockReq.EXPECT().WebchatV1MessageCreate(ctx, customerID, sessionID, wcmessage.DirectionInbound, uuid.Nil, "hello").Return(responseMessage, nil)

	res, err := h.WebchatMessageCreate(ctx, a, sessionID, wcmessage.DirectionInbound, "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := responseMessage.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatMessageCreate_Direct_CrossTenant_WrongWidgetScope is a
// regression test for a cross-tenant IDOR: a direct-scope JWT issued for
// widget A must NOT be able to inject a message into a session that
// belongs to a different widget B (e.g. by guessing/enumerating a
// session_id UUID). Before the fix, WebchatMessageCreate's direct-token
// branch only checked HasAllowedResourceType("webchat_session") and never
// verified the resolved session's WidgetID against DirectScope.ResourceID.
func Test_WebchatMessageCreate_Direct_CrossTenant_WrongWidgetScope(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	attackerScopedWidgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	victimWidgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	victimSessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "webchat_widget",
		ResourceID:           attackerScopedWidgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	// The victim session genuinely belongs to a different widget/customer.
	victimSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: victimSessionID, CustomerID: victimCustomerID},
		WidgetID: victimWidgetID,
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, victimSessionID).Return(victimSession, nil)

	if _, err := h.WebchatMessageCreate(ctx, a, victimSessionID, wcmessage.DirectionInbound, "injected"); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatSessionEnd_Direct_CrossTenant_WrongWidgetScope is the
// analogous regression test for WebchatSessionEnd: a visitor JWT scoped to
// widget A must not be able to forcibly end a session belonging to widget B.
func Test_WebchatSessionEnd_Direct_CrossTenant_WrongWidgetScope(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	attackerScopedWidgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	victimWidgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	victimSessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "webchat_widget",
		ResourceID:           attackerScopedWidgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	victimSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: victimSessionID, CustomerID: victimCustomerID},
		WidgetID: victimWidgetID,
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, victimSessionID).Return(victimSession, nil)

	if _, err := h.WebchatSessionEnd(ctx, a, victimSessionID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatSessionEnd_Direct verifies the legitimate visitor-initiated
// end still works once the session's widget matches the JWT's scope.
func Test_WebchatSessionEnd_Direct(t *testing.T) {
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

	session := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
	}
	ended := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   wcsession.StatusEnded,
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(session, nil)
	mockReq.EXPECT().WebchatV1SessionEnd(ctx, sessionID).Return(ended, nil)

	res, err := h.WebchatSessionEnd(ctx, a, sessionID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := ended.ConvertWebhookMessage()
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}
