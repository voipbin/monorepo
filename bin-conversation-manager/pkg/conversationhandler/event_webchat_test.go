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
)

// Test_EventWebchat_Inbound verifies an inbound webchat_message_created
// event resolves/creates a Conversation with Self=Widget address,
// Peer=Session address (both commonaddress.TypeWebchat, no Account
// created anywhere in this path), creates the mirrored message with
// ReferenceType=webchat, and does NOT trigger any Flow -- runExecuteModeFlow's
// default case is a structural no-op for conversation.TypeWebchat.
func Test_EventWebchat_Inbound(t *testing.T) {
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
		// OwnerType/OwnerID zero-value -> getExecuteMode returns
		// ExecuteModeFlow, which is the branch under test here (proves
		// the flow-trigger path is a structural no-op, not merely
		// "untested").
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

	// CRITICAL assertion for B안 (design doc §16.5): no FlowV1ActiveflowCreate
	// call is expected on the mockReq. If runExecuteModeFlow's default case
	// ever stops being a no-op for TypeWebchat, gomock fails this test with
	// "unexpected call" the moment that RPC fires.

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
