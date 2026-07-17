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

	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcsession "monorepo/bin-webchat-manager/models/session"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/models/auth"
)

// Test_WebchatMessageList_NoFilter verifies the existing behavior is
// unchanged when sessionID is uuid.Nil (no filter requested): only
// customer_id/deleted are applied, and sessionGet is never called.
func Test_WebchatMessageList_NoFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	messageID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseMessages := []*wcmessage.Message{
		{Identity: commonidentity.Identity{ID: messageID, CustomerID: customerID}},
	}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1MessageList(ctx, "2026-01-01 00:00:00.000000", uint64(10), map[wcmessage.Field]any{
		wcmessage.FieldCustomerID: customerID,
		wcmessage.FieldDeleted:    false,
	}).Return(responseMessages, nil)

	res, err := h.WebchatMessageList(ctx, a, 10, "", uuid.Nil)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := []*wcmessage.WebhookMessage{responseMessages[0].ConvertWebhookMessage()}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatMessageList_SessionFilter verifies that a non-nil sessionID
// (a) is validated for ownership via sessionGet and (b) is added to the
// filter map passed to WebchatV1MessageList.
func Test_WebchatMessageList_SessionFilter(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: customerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	responseSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
	}
	responseMessages := []*wcmessage.Message{}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(responseSession, nil)
	mockReq.EXPECT().WebchatV1MessageList(ctx, "2026-01-01 00:00:00.000000", uint64(10), map[wcmessage.Field]any{
		wcmessage.FieldCustomerID: customerID,
		wcmessage.FieldDeleted:    false,
		wcmessage.FieldSessionID:  sessionID,
	}).Return(responseMessages, nil)

	res, err := h.WebchatMessageList(ctx, a, 10, "", sessionID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("Wrong match. expect: empty, got: %v", res)
	}
}

// Test_WebchatMessageList_SessionFilter_CrossTenant is a regression guard:
// a caller must not be able to pass another customer's session_id to
// enumerate that customer's messages, even though the caller's own
// hasPermission check against a.CustomerID trivially passes.
func Test_WebchatMessageList_SessionFilter_CrossTenant(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	callerCustomerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	victimCustomerID := uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000ff")
	victimSessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: callerCustomerID,
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	victimSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: victimSessionID, CustomerID: victimCustomerID},
	}

	mockUtil.EXPECT().TimeGetCurTime().Return("2026-01-01 00:00:00.000000")
	mockReq.EXPECT().WebchatV1SessionGet(ctx, victimSessionID).Return(victimSession, nil)

	if _, err := h.WebchatMessageList(ctx, a, 10, "", victimSessionID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatMessageList_Direct verifies the new direct-scope (visitor)
// path: a valid direct JWT scoped to the session's own widget can list
// that session's messages, filtered by the SESSION's actual customer_id
// (not the JWT's own DirectScope.CustomerID, defense-in-depth mirroring
// WebchatMessageCreate's ownerCustomerID pattern -- VOIP-1265 §9.2).
func Test_WebchatMessageList_Direct(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

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
	responseWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
	}
	responseMessages := []*wcmessage.Message{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("cc847807-6cc4-4713-9dec-53a42840e74c"), CustomerID: customerID}},
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(responseSession, nil)
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(responseWidget, nil)
	mockReq.EXPECT().WebchatV1MessageList(ctx, gomock.Any(), uint64(10), map[wcmessage.Field]any{
		wcmessage.FieldCustomerID: customerID,
		wcmessage.FieldDeleted:    false,
		wcmessage.FieldSessionID:  sessionID,
	}).Return(responseMessages, nil)

	res, err := h.WebchatMessageList(ctx, a, 10, "", sessionID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := []*wcmessage.WebhookMessage{responseMessages[0].ConvertWebhookMessage()}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", expectRes, res)
	}
}

// Test_WebchatMessageList_Direct_NoSessionID verifies a direct-scope
// caller omitting session_id is rejected -- there is no "list all my
// messages across sessions" concept for a visitor.
func Test_WebchatMessageList_Direct_NoSessionID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
	}
	ctx := context.Background()

	widgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		ResourceType:         "webchat_widget",
		ResourceID:           widgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	if _, err := h.WebchatMessageList(ctx, a, 10, "", uuid.Nil); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatMessageList_Direct_WrongWidget is a regression guard: a
// visitor's direct-scope JWT for widget A must not be able to list
// messages for a session belonging to widget B, even within the same
// customer.
func Test_WebchatMessageList_Direct_WrongWidget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	callerWidgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	otherWidgetID := uuid.FromStringOrNil("dd847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	a := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID:           customerID,
		ResourceType:         "webchat_widget",
		ResourceID:           callerWidgetID,
		AllowedResourceTypes: []string{"webchat_session"},
	})

	otherWidgetSession := &wcsession.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: otherWidgetID,
	}

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(otherWidgetSession, nil)

	if _, err := h.WebchatMessageList(ctx, a, 10, "", sessionID); err == nil {
		t.Error("Wrong match. expect: permission denied error, got: ok")
	}
}

// Test_WebchatMessageList_Direct_DeletedWidget is a regression guard for
// the exact CRUD-asymmetry bug class VOIP-1265 fixes: a visitor holding
// a direct-scope JWT for a now soft-deleted widget must not be able to
// read message history via List, mirroring WebchatMessageCreate's
// direct branch which already rejects Create on a deleted widget.
func Test_WebchatMessageList_Direct_DeletedWidget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &serviceHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		reqHandler:  mockReq,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	widgetID := uuid.FromStringOrNil("bb847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

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

	mockReq.EXPECT().WebchatV1SessionGet(ctx, sessionID).Return(responseSession, nil)
	// widgetGet treats a soft-deleted widget as not-found (see
	// webchat_widget.go's widgetGet, mirroring flowGet's pattern) -- a
	// TMDelete-set widget causes WebchatV1WidgetGet's caller-side
	// wrapper to return ErrNotFound. Simulate that directly since
	// widgetGet's soft-delete check runs on the *returned* widget, not
	// inside the mock.
	deletedWidget := &wcwidget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		TMDelete: &deletedAt,
	}
	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(deletedWidget, nil)

	if _, err := h.WebchatMessageList(ctx, a, 10, "", sessionID); err == nil {
		t.Error("Wrong match. expect: not found error (soft-deleted widget), got: ok")
	}
}

var deletedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
