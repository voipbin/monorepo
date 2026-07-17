package conversationhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
)

// Test_CreateAndExecuteFlow_Normal verifies a fresh Conversation is
// created (no dedup lookup) and its activeflow is Create+Executed with
// ReferenceType=Conversation. Used by bin-webchat-manager's
// sessionhandler.Create at webchat session-create time to trigger
// Widget.SessionFlowID.
func Test_CreateAndExecuteFlow_Normal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &conversationHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	conversationID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")
	activeflowID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "widget-id"}
	peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "session-id"}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     self,
		Peer:     peer,
	}

	mockUtil.EXPECT().UUIDCreate().Return(conversationID)
	mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(cv, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, conversation.EventTypeConversationCreated, cv)

	expectVariables := map[string]string{
		variableConversationSelfName:       cv.Self.Name,
		variableConversationSelfDetail:     cv.Self.Detail,
		variableConversationSelfTarget:     cv.Self.Target,
		variableConversationSelfTargetName: cv.Self.TargetName,
		variableConversationSelfType:       string(cv.Self.Type),

		variableConversationPeerName:       cv.Peer.Name,
		variableConversationPeerDetail:     cv.Peer.Detail,
		variableConversationPeerTarget:     cv.Peer.Target,
		variableConversationPeerTargetName: cv.Peer.TargetName,
		variableConversationPeerType:       string(cv.Peer.Type),

		variableConversationID:      cv.ID.String(),
		variableConversationOwnerID: cv.OwnerID.String(),
	}

	callCreate := mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		customerID,
		flowID,
		fmactiveflow.ReferenceTypeConversation,
		conversationID,
		uuid.Nil,
		nil,
		"",
		fmactiveflow.WebhookMethodNone,
	).Return(&fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{ID: activeflowID},
	}, nil)
	callSetVariable := mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, expectVariables).Return(nil)
	callExecute := mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Return(nil)
	gomock.InOrder(callCreate, callSetVariable, callExecute)

	res, err := h.CreateAndExecuteFlow(ctx, customerID, flowID, conversation.TypeWebchat, "", self, peer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != conversationID {
		t.Errorf("Wrong match. expect: %s, got: %s", conversationID, res.ID)
	}
}

// Test_CreateAndExecuteFlow_NoFlowID verifies the Conversation is still
// created when flowID is uuid.Nil, but no activeflow is triggered.
func Test_CreateAndExecuteFlow_NoFlowID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &conversationHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	conversationID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "widget-id"}
	peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "session-id"}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     self,
		Peer:     peer,
	}

	mockUtil.EXPECT().UUIDCreate().Return(conversationID)
	mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(cv, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, conversation.EventTypeConversationCreated, cv)

	// No FlowV1ActiveflowCreate/Execute expected -- their absence from
	// the mock is enforced by gomock failing on any unexpected call.

	res, err := h.CreateAndExecuteFlow(ctx, customerID, uuid.Nil, conversation.TypeWebchat, "", self, peer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != conversationID {
		t.Errorf("Wrong match. expect: %s, got: %s", conversationID, res.ID)
	}
}

// Test_CreateAndExecuteFlow_ActiveflowCreateFails verifies the
// Conversation is still returned (best-effort) when the activeflow
// trigger itself fails -- the Conversation was already committed.
func Test_CreateAndExecuteFlow_ActiveflowCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &conversationHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	conversationID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "widget-id"}
	peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "session-id"}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     self,
		Peer:     peer,
	}

	mockUtil.EXPECT().UUIDCreate().Return(conversationID)
	mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(cv, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, conversation.EventTypeConversationCreated, cv)

	mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeConversation, conversationID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone,
	).Return(nil, context.DeadlineExceeded)

	res, err := h.CreateAndExecuteFlow(ctx, customerID, flowID, conversation.TypeWebchat, "", self, peer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok (best-effort), got: %v", err)
	}
	if res.ID != conversationID {
		t.Errorf("Wrong match. expect: %s, got: %s", conversationID, res.ID)
	}
}

// Test_CreateAndExecuteFlow_SetVariablesFails verifies the Conversation
// is still returned (best-effort) when setting the voipbin.conversation.*
// flow variables fails, and that FlowV1ActiveflowExecute is never called
// in that case (variables must be set before execution starts).
func Test_CreateAndExecuteFlow_SetVariablesFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &conversationHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	flowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	conversationID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")
	activeflowID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")

	self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "widget-id"}
	peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: "session-id"}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: conversationID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     self,
		Peer:     peer,
	}

	mockUtil.EXPECT().UUIDCreate().Return(conversationID)
	mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ConversationGet(ctx, conversationID).Return(cv, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, conversation.EventTypeConversationCreated, cv)

	callCreate := mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx, uuid.Nil, customerID, flowID, fmactiveflow.ReferenceTypeConversation, conversationID, uuid.Nil, nil, "", fmactiveflow.WebhookMethodNone,
	).Return(&fmactiveflow.Activeflow{
		Identity: commonidentity.Identity{ID: activeflowID},
	}, nil)
	callSetVariable := mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, gomock.Any()).Return(context.DeadlineExceeded)
	gomock.InOrder(callCreate, callSetVariable)
	// FlowV1ActiveflowExecute must NOT be called -- its absence from the
	// mock is enforced by gomock failing on any unexpected call.

	res, err := h.CreateAndExecuteFlow(ctx, customerID, flowID, conversation.TypeWebchat, "", self, peer)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok (best-effort), got: %v", err)
	}
	if res.ID != conversationID {
		t.Errorf("Wrong match. expect: %s, got: %s", conversationID, res.ID)
	}
}
