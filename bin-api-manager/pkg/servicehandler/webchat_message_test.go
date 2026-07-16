package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	amagent "monorepo/bin-agent-manager/models/agent"

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

// Test_WebchatMessageCreate_Agent_CrossTenant_WrongSessionOwner is a
// regression test for a cross-tenant message-injection bug: an
// agent/accesskey with admin/manager permission on their OWN customer
// account must not be able to post a message into a session_id belonging
// to a DIFFERENT customer's widget -- the reply would be delivered to that
// other customer's live visitor as if sent by the caller's own agent.
// Before the fix, the a.IsAgent()||a.IsAccesskey() branch only checked
// hasPermission(ctx, a, a.CustomerID, ...) -- a tautology that never
// resolved the session or verified it belonged to the caller's customer.
func Test_WebchatMessageCreate_Agent_CrossTenant_WrongSessionOwner(t *testing.T) {
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
	victimSessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	victimSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: victimSessionID, CustomerID: victimCustomerID},
		WidgetID: victimWidgetID,
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, victimSessionID).Return(victimSession, nil)

	if _, err := h.WebchatMessageCreate(ctx, a, victimSessionID, wcmessage.DirectionOutbound, "injected reply"); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatMessageCreate_Direct_DirectionSpoofIgnored is a regression
// test for message-direction spoofing (Round 3 finding): a visitor holding
// only a direct-scope JWT must never be able to author an
// outbound (agent-authored-looking) message, even if the client explicitly
// requests direction=outbound in the API call. The server must silently
// force direction=inbound for this auth path regardless of the caller-
// supplied value -- verified here by asserting the downstream RPC call is
// made with DirectionInbound even though the handler was invoked with
// DirectionOutbound.
func Test_WebchatMessageCreate_Direct_DirectionSpoofIgnored(t *testing.T) {
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
		Text:      "spoof attempt",
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(responseSession, nil)
	// The mock strictly asserts DirectionInbound -- if the handler forwarded
	// the caller-supplied DirectionOutbound instead, gomock would reject
	// the call as unexpected and this test would fail.
	mockReq.EXPECT().WebchatV1MessageCreate(ctx, customerID, sessionID, wcmessage.DirectionInbound, uuid.Nil, "spoof attempt").Return(responseMessage, nil)

	// Caller explicitly requests DirectionOutbound -- must be overridden.
	if _, err := h.WebchatMessageCreate(ctx, a, sessionID, wcmessage.DirectionOutbound, "spoof attempt"); err != nil {
		t.Fatalf("Wrong match. expect: ok (direction silently overridden), got: %v", err)
	}
}

// Test_WebchatMessageCreate_Agent_DirectionSpoofIgnored is the symmetric
// regression test: an authenticated agent/accesskey must never be able to
// author an inbound (visitor-authored-looking) message, even if the
// client explicitly requests direction=inbound.
func Test_WebchatMessageCreate_Agent_DirectionSpoofIgnored(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9f5e-4762c3c1e4e5")
	agentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	session := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
	}
	responseMessage := &wcmessage.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: wcmessage.DirectionOutbound,
		SenderID:  agentID,
		Text:      "spoof attempt",
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(session, nil)
	mockReq.EXPECT().WebchatV1MessageCreate(ctx, customerID, sessionID, wcmessage.DirectionOutbound, agentID, "spoof attempt").Return(responseMessage, nil)

	// Caller explicitly requests DirectionInbound -- must be overridden.
	if _, err := h.WebchatMessageCreate(ctx, a, sessionID, wcmessage.DirectionInbound, "spoof attempt"); err != nil {
		t.Fatalf("Wrong match. expect: ok (direction silently overridden), got: %v", err)
	}
}

// Test_WebchatMessageCreate_Agent_SuperAdmin_CrossTenant_OwnerCustomerID is
// a regression test for a customer_id data-integrity bug: hasPermission
// short-circuits to true for PermissionProjectSuperAdmin regardless of
// a.CustomerID vs the session's actual owner. Before this fix,
// WebchatMessageCreate unconditionally passed the caller's OWN a.CustomerID
// (the superadmin's, not the session/widget owner's) to
// WebchatV1MessageCreate -- creating a message tagged with the WRONG
// customer_id, invisible to the session owner's own
// WebchatMessageList/Get calls. Verified here by asserting the downstream
// RPC call is made with the SESSION's customer_id, not the superadmin
// caller's.
func Test_WebchatMessageCreate_Agent_SuperAdmin_CrossTenant_OwnerCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	superAdminCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	sessionOwnerCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9f5e-4762c3c1e4e5")
	superAdminAgentID := uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         superAdminAgentID,
			CustomerID: superAdminCustomerID,
		},
		Permission: amagent.PermissionProjectSuperAdmin,
	})

	session := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: sessionOwnerCustomerID},
	}
	responseMessage := &wcmessage.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: sessionOwnerCustomerID},
		SessionID: sessionID,
		Direction: wcmessage.DirectionOutbound,
		SenderID:  superAdminAgentID,
		Text:      "reply",
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(session, nil)
	// The mock strictly asserts sessionOwnerCustomerID -- if the handler
	// forwarded the superadmin's own a.CustomerID instead, gomock would
	// reject the call as unexpected and this test would fail.
	mockReq.EXPECT().WebchatV1MessageCreate(ctx, sessionOwnerCustomerID, sessionID, wcmessage.DirectionOutbound, superAdminAgentID, "reply").Return(responseMessage, nil)

	if _, err := h.WebchatMessageCreate(ctx, a, sessionID, wcmessage.DirectionOutbound, "reply"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}
