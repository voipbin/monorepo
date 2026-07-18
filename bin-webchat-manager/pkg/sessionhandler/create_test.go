package sessionhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
	"monorepo/bin-webchat-manager/pkg/widgethandler"
)

// Test_Create_NoSessionFlowConfigured verifies session creation when
// the Widget has no SessionFlowID configured: no conversation-manager
// RPC call is made, and no welcome delivery of any kind is attached
// to the response (accepted no-greeting outcome — see design doc
// 2026-07-18-webchat-welcome-message-flow-consolidation-design.md §9).
func Test_Create_NoSessionFlowConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}

	w := &widget.Widget{
		Identity:      commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		SessionFlowID: uuid.Nil,
	}

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockWidget := widgethandler.NewMockWidgetHandler(mc)
	h := &sessionHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		widgetHandler: mockWidget,
	}
	ctx := context.Background()

	mockUtil.EXPECT().UUIDCreate().Return(sessionID)
	mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockWidget.EXPECT().Get(ctx, widgetID).Return(w, nil)

	// No ConversationV1ConversationCreateAndExecuteFlow, no
	// SessionUpdate expected -- their absence from the mock is
	// enforced by gomock failing on any unexpected call.

	res, err := h.Create(ctx, customerID, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	// expectRes is a value copy, independent of sess -- SessionGet's mock
	// returns the sess pointer, so if Create() ever mutated the pointee
	// in place (as it did with WelcomeMessage before), a DeepEqual against
	// sess itself would be tautological (the mutation lands in both
	// operands). Comparing against a separately-constructed value ensures
	// this test genuinely fails on any post-fetch mutation.
	expectRes := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

// Test_Create_SessionFlowConfigured_TriggersFlow verifies session
// creation when the Widget has a SessionFlowID configured: the
// conversation-manager CreateAndExecuteFlow RPC is called with a
// self=Widget.ID/peer=Session.ID address pair, and Session.ActiveflowID
// is recorded from the resulting Conversation.
func Test_Create_SessionFlowConfigured_TriggersFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionFlowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	conversationID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}

	w := &widget.Widget{
		Identity:      commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		SessionFlowID: sessionFlowID,
	}

	cv := &cvconversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
	}

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWidget := widgethandler.NewMockWidgetHandler(mc)
	h := &sessionHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    mockReq,
		widgetHandler: mockWidget,
	}
	ctx := context.Background()

	mockUtil.EXPECT().UUIDCreate().Return(sessionID)
	mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockWidget.EXPECT().Get(ctx, widgetID).Return(w, nil)

	mockReq.EXPECT().ConversationV1ConversationCreateAndExecuteFlow(
		ctx,
		customerID,
		sessionFlowID,
		cvconversation.TypeWebchat,
		"",
		commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()},
		commonaddress.Address{Type: commonaddress.TypeWebchat, Target: sessionID.String()},
	).Return(cv, nil)

	mockDB.EXPECT().SessionUpdate(ctx, sessionID, map[session.Field]any{
		session.FieldActiveflowID: cv.ID,
	}).Return(nil)

	res, err := h.Create(ctx, customerID, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, sess) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", sess, res)
	}
}

// Test_Create_WidgetFetchFails_SessionStillSucceeds verifies a Widget
// fetch failure does not fail Session creation -- best-effort.
func Test_Create_WidgetFetchFails_SessionStillSucceeds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	sessionID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockWidget := widgethandler.NewMockWidgetHandler(mc)
	h := &sessionHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		widgetHandler: mockWidget,
	}
	ctx := context.Background()

	mockUtil.EXPECT().UUIDCreate().Return(sessionID)
	mockDB.EXPECT().SessionCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockWidget.EXPECT().Get(ctx, widgetID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Create(ctx, customerID, widgetID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, sess) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", sess, res)
	}
}
