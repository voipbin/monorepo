package conversationhandler

import (
	"context"
	"encoding/json"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"
	wcwidget "monorepo/bin-webchat-manager/models/widget"
)

// Test_EventWebchat_Inbound_MessageFlowConfigured verifies an inbound
// webchat_message_created event resolves/creates a Conversation with
// Self=Widget address, Peer=Session address (both
// commonaddress.TypeWebchat, no Account created anywhere in this
// path), creates the mirrored message with ReferenceType=webchat, and
// -- per the message-flow-owner-migration design -- DOES trigger a
// Flow via runExecuteModeFlowWebchat when the Widget has
// MessageFlowID configured, with ReferenceType=ReferenceTypeConversation
// (not ReferenceTypeWebchat) and ReferenceID=cv.ID.
func Test_EventWebchat_Inbound_MessageFlowConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		messageHandler: mockMessage,
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	convID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")
	messageFlowID := uuid.FromStringOrNil("2b5bc824-2066-11f0-81b0-672de53dec30")
	activeflowID := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")

	wm := webchatMessage{
		ID:         msgID,
		CustomerID: customerID,
		WidgetID:   widgetID,
		SessionID:  sessionID,
		Direction:  webchatDirectionInbound,
		Text:       "hello there",
	}
	data, errMarshal := json.Marshal(wm)
	if errMarshal != nil {
		t.Fatalf("could not marshal test fixture: %v", errMarshal)
	}

	expectSelf := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
	expectPeer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: sessionID.String()}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: convID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     expectSelf,
		Peer:     expectPeer,
		// OwnerType/OwnerID zero-value -> getExecuteMode returns
		// ExecuteModeFlow, which is the branch under test here.
	}

	convMsg := &message.Message{
		Identity:       commonidentity.Identity{ID: msgID, CustomerID: customerID},
		ConversationID: convID,
		Direction:      message.DirectionIncoming,
		ReferenceType:  message.ReferenceTypeWebchat,
		Text:           "hello there",
	}

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, expectSelf, expectPeer).Return(cv, nil)
	mockMessage.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, args messagehandler.MessageCreateArgs) (*message.Message, error) {
			if args.ReferenceType != message.ReferenceTypeWebchat {
				t.Errorf("expected ReferenceType=webchat, got: %v", args.ReferenceType)
			}
			if args.Direction != message.DirectionIncoming {
				t.Errorf("expected Direction=incoming, got: %v", args.Direction)
			}
			return convMsg, nil
		},
	)

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(&wcwidget.Widget{
		Identity:      commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		MessageFlowID: messageFlowID,
	}, nil)

	// CRITICAL assertion for the message-flow-owner-migration design:
	// ReferenceType=ReferenceTypeConversation (NOT ReferenceTypeWebchat),
	// ReferenceID=cv.ID (NOT sessionID) -- this is the key behavioral
	// change from the prior "B안" design.
	mockReq.EXPECT().FlowV1ActiveflowCreate(
		ctx,
		uuid.Nil,
		convMsg.CustomerID,
		messageFlowID,
		fmactiveflow.ReferenceTypeConversation,
		convID,
		uuid.Nil,
		nil,
		gomock.Any(),
		gomock.Any(),
	).Return(&fmactiveflow.Activeflow{Identity: commonidentity.Identity{ID: activeflowID}}, nil)
	mockReq.EXPECT().FlowV1VariableSetVariable(ctx, activeflowID, gomock.Any()).Times(2).Return(nil)
	mockReq.EXPECT().FlowV1ActiveflowExecute(ctx, activeflowID).Return(nil)

	if err := h.eventWebchat(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_EventWebchat_Inbound_NoMessageFlowConfigured verifies an inbound
// webchat_message_created event on a Widget with NO MessageFlowID
// configured resolves/creates the Conversation and Message as usual,
// but triggers no Flow -- executeActiveflow's own no-op-on-uuid.Nil
// behavior, reached via runExecuteModeFlowWebchat.
func Test_EventWebchat_Inbound_NoMessageFlowConfigured(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		messageHandler: mockMessage,
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	convID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	wm := webchatMessage{
		ID:         msgID,
		CustomerID: customerID,
		WidgetID:   widgetID,
		SessionID:  sessionID,
		Direction:  webchatDirectionInbound,
		Text:       "hello there",
	}
	data, errMarshal := json.Marshal(wm)
	if errMarshal != nil {
		t.Fatalf("could not marshal test fixture: %v", errMarshal)
	}

	expectSelf := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
	expectPeer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: sessionID.String()}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: convID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     expectSelf,
		Peer:     expectPeer,
	}

	convMsg := &message.Message{
		Identity:       commonidentity.Identity{ID: msgID, CustomerID: customerID},
		ConversationID: convID,
		Direction:      message.DirectionIncoming,
		ReferenceType:  message.ReferenceTypeWebchat,
		Text:           "hello there",
	}

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, expectSelf, expectPeer).Return(cv, nil)
	mockMessage.EXPECT().Create(ctx, gomock.Any()).Return(convMsg, nil)

	mockReq.EXPECT().WebchatV1WidgetGet(ctx, widgetID).Return(&wcwidget.Widget{
		Identity:      commonidentity.Identity{ID: widgetID, CustomerID: customerID},
		MessageFlowID: uuid.Nil,
	}, nil)

	// No FlowV1ActiveflowCreate/Execute expected -- executeActiveflow
	// no-ops when flowID == uuid.Nil.

	if err := h.eventWebchat(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_EventWebchat_Outbound verifies an outbound webchat_message_created
// event (agent reply or Flow/AI response) resolves the same Conversation
// with Self/Peer unchanged (still Widget=self, Session=peer -- outbound
// does not swap them, since DeriveEndpoints handles the direction-based
// source/destination swap internally) and creates the mirrored message.
func Test_EventWebchat_Outbound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		reqHandler:    mockReq,

		messageHandler: mockMessage,
	}

	ctx := context.Background()

	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	widgetID := uuid.FromStringOrNil("aa847807-6cc4-4713-9dec-53a42840e74c")
	sessionID := uuid.FromStringOrNil("876defde-ad5e-11ed-a8c3-7bc19647b03f")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	convID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	wm := webchatMessage{
		ID:         msgID,
		CustomerID: customerID,
		WidgetID:   widgetID,
		SessionID:  sessionID,
		Direction:  webchatDirectionOutbound,
		Text:       "how can I help?",
	}
	data, errMarshal := json.Marshal(wm)
	if errMarshal != nil {
		t.Fatalf("could not marshal test fixture: %v", errMarshal)
	}

	expectSelf := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
	expectPeer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: sessionID.String()}

	cv := &conversation.Conversation{
		Identity: commonidentity.Identity{ID: convID, CustomerID: customerID},
		Type:     conversation.TypeWebchat,
		Self:     expectSelf,
		Peer:     expectPeer,
	}

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, expectSelf, expectPeer).Return(cv, nil)
	mockMessage.EXPECT().Get(ctx, msgID).Return(nil, dbhandler.ErrNotFound)
	mockMessage.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(
		func(_ context.Context, args messagehandler.MessageCreateArgs) (*message.Message, error) {
			if args.ReferenceType != message.ReferenceTypeWebchat {
				t.Errorf("expected ReferenceType=webchat, got: %v", args.ReferenceType)
			}
			if args.Direction != message.DirectionOutgoing {
				t.Errorf("expected Direction=outgoing, got: %v", args.Direction)
			}
			return &message.Message{Identity: commonidentity.Identity{ID: msgID}}, nil
		},
	)

	// No FlowV1ActiveflowCreate expected either -- outbound never triggers
	// a Flow regardless of ExecuteMode (eventWebchat's outbound path never
	// even calls getExecuteMode).

	if err := h.eventWebchat(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_EventWebchat_UnknownDirection verifies a malformed/unknown
// direction is rejected rather than silently treated as inbound or
// outbound.
func Test_EventWebchat_UnknownDirection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &conversationHandler{}
	ctx := context.Background()

	wm := webchatMessage{Direction: "sideways"}
	data, _ := json.Marshal(wm)

	if err := h.eventWebchat(ctx, data); err == nil {
		t.Error("Wrong match. expect: error, got: ok")
	}
}
