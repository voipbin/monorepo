package subscribehandler

import (
	"encoding/json"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

func Test_NewSubscribeHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)

	subscribeQueue := "test-queue"
	subscribeTargets := []string{"target1", "target2"}

	h := NewSubscribeHandler(
		mockSock,
		subscribeQueue,
		subscribeTargets,
		mockAccount,
		mockConversation,
	)

	if h == nil {
		t.Errorf("Expected non-nil SubscribeHandler, got nil")
	}
}

// Test_processEventRun is skipped because it launches a goroutine
// which makes it difficult to test reliably with mocks

func Test_processEvent_MessageManagerMessageCreated(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)

	h := &subscribeHandler{
		sockHandler:         mockSock,
		accountHandler:      mockAccount,
		conversationHandler: mockConversation,
	}

	msg := &mmmessage.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
		},
		Text: "test message",
	}

	data, _ := json.Marshal(msg)

	event := &commonsock.Event{
		Publisher: publisherMessageManager,
		Type:      string(mmmessage.EventTypeMessageCreated),
		Data:      json.RawMessage(data),
	}

	mockConversation.EXPECT().Event(gomock.Any(), conversation.TypeMessage, gomock.Any()).Return(nil)

	h.processEvent(event)
}

func Test_processEvent_UnknownEvent(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)

	h := &subscribeHandler{
		sockHandler:         mockSock,
		accountHandler:      mockAccount,
		conversationHandler: mockConversation,
	}

	event := &commonsock.Event{
		Publisher: "unknown-publisher",
		Type:      "unknown-type",
		Data:      json.RawMessage("{}"),
	}

	// Should not panic for unknown events, just return
	h.processEvent(event)
}
