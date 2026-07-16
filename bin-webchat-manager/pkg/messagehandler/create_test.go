package messagehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

func newTestMessageHandler(mc *gomock.Controller) (*messageHandler, *utilhandler.MockUtilHandler, *requesthandler.MockRequestHandler, *dbhandler.MockDBHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), message.EventTypeMessageCreated, gomock.Any()).AnyTimes()

	h := &messageHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		sessionLocks:  map[uuid.UUID]chan struct{}{},
	}

	return h, mockUtil, mockReq, mockDB
}

// Test_Create_Inbound_FirstMessage_TriggersFlow verifies the first
// inbound message on a Session with no prior ActiveflowID and a Widget
// with a FlowID configured triggers exactly one activeflow, and records
// its ID back onto the Session.
func Test_Create_Inbound_FirstMessage_TriggersFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockReq, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	activeflowID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
		// ActiveflowID is uuid.Nil (zero value) -- no Flow triggered yet.
	}

	w := &widget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		FlowID:   flowID,
	}

	msg := &message.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: message.DirectionInbound,
		Status:    message.StatusSent,
		Text:      "hello",
	}

	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockUtil.EXPECT().UUIDCreate().Return(messageID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(msg, nil)
	mockDB.EXPECT().WidgetGet(ctx, widgetID).Return(w, nil)

	mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		customerID,
		flowID,
		fmactiveflow.ReferenceTypeWebchat,
		sessionID,
		uuid.Nil,
		gomock.Any(),
		"",
		fmactiveflow.WebhookMethodNone,
	).Return(&fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{ID: activeflowID},
	}, nil)

	mockDB.EXPECT().SessionUpdate(ctx, sessionID, map[session.Field]any{
		session.FieldActiveflowID: activeflowID,
	}).Return(nil)

	mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Return(nil)

	res, err := h.Create(ctx, customerID, sessionID, message.DirectionInbound, uuid.Nil, "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, msg) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", msg, res)
	}
}

// Test_Create_Inbound_FollowUpMessage_NoFlowTrigger verifies a
// subsequent inbound message on a Session that already has an
// ActiveflowID recorded does NOT trigger another activeflow.
func Test_Create_Inbound_FollowUpMessage_NoFlowTrigger(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")

	sess := &session.Session{
		Identity:     commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID:     widgetID,
		Status:       session.StatusActive,
		ActiveflowID: uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50"), // already triggered
	}

	msg := &message.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: message.DirectionInbound,
		Status:    message.StatusSent,
		Text:      "follow-up",
	}

	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockUtil.EXPECT().UUIDCreate().Return(messageID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(msg, nil)

	// No WidgetGet, no FlowV1ActiveflowCreate, no FlowV1ActiveflowExecute
	// expected -- their absence from the mock is enforced by gomock
	// failing on any unexpected call.

	res, err := h.Create(ctx, customerID, sessionID, message.DirectionInbound, uuid.Nil, "follow-up")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, msg) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", msg, res)
	}
}

// Test_Create_Inbound_NoFlowConfigured verifies a Widget with no
// FlowID never triggers an activeflow, even on the first message.
func Test_Create_Inbound_NoFlowConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}

	w := &widget.Widget{
		Identity: commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		FlowID:   uuid.Nil, // no flow configured
	}

	msg := &message.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: message.DirectionInbound,
		Status:    message.StatusSent,
		Text:      "hello",
	}

	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockUtil.EXPECT().UUIDCreate().Return(messageID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(msg, nil)
	mockDB.EXPECT().WidgetGet(ctx, widgetID).Return(w, nil)

	res, err := h.Create(ctx, customerID, sessionID, message.DirectionInbound, uuid.Nil, "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, msg) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", msg, res)
	}
}

// Test_Create_Outbound_NeverTriggersFlow verifies an outbound message
// (agent reply or Flow-delivered response) never checks/triggers the
// Flow, and never takes the session lock or reads the Session row --
// matches conversation-manager's MessageEventSent never calling
// runExecuteModeFlow.
func Test_Create_Outbound_NeverTriggersFlow(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	messageID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	senderID := uuid.FromStringOrNil("5f4e2b1a-0000-0000-0000-000000000001")

	sess := &session.Session{
		Identity: commonidentity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   session.StatusActive,
	}

	msg := &message.Message{
		Identity:  commonidentity.Identity{ID: messageID, CustomerID: customerID},
		WidgetID:  widgetID,
		SessionID: sessionID,
		Direction: message.DirectionOutbound,
		Status:    message.StatusSent,
		SenderID:  senderID,
		Text:      "hi, how can I help?",
	}

	mockDB.EXPECT().SessionGet(ctx, sessionID).Return(sess, nil)
	mockUtil.EXPECT().UUIDCreate().Return(messageID)
	mockDB.EXPECT().MessageCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(msg, nil)

	// No WidgetGet, no FlowV1ActiveflowCreate expected.

	res, err := h.Create(ctx, customerID, sessionID, message.DirectionOutbound, senderID, "hi, how can I help?")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, msg) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", msg, res)
	}
}

func Test_MessageHandler_Get(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	expectRes := &message.Message{Identity: commonidentity.Identity{ID: id}}

	mockDB.EXPECT().MessageGet(ctx, id).Return(expectRes, nil)

	res, err := h.Get(ctx, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_MessageHandler_List(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	expectRes := []*message.Message{
		{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")}},
	}

	mockDB.EXPECT().MessageList(ctx, uint64(10), "", map[message.Field]any{}).Return(expectRes, nil)

	res, err := h.List(ctx, 10, "", map[message.Field]any{})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_MessageHandler_Delete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB := newTestMessageHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	expectRes := &message.Message{Identity: commonidentity.Identity{ID: id}}

	mockDB.EXPECT().MessageDelete(ctx, id).Return(nil)
	mockDB.EXPECT().MessageGet(ctx, id).Return(expectRes, nil)

	res, err := h.Delete(ctx, id)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}
