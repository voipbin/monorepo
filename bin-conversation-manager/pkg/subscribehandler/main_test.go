package subscribehandler

import (
	"encoding/json"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	emmemail "monorepo/bin-email-manager/models/email"
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

func Test_processEvent_EmailManagerEmailCreated(t *testing.T) {
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

	e := &emmemail.Email{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440011"),
			CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440012"),
		},
		Subject: "test subject",
		Content: "test body",
	}

	data, _ := json.Marshal(e)

	event := &commonsock.Event{
		Publisher: publisherEmailManager,
		Type:      emmemail.EventTypeCreated,
		Data:      json.RawMessage(data),
	}

	mockConversation.EXPECT().EmailEventSent(gomock.Any(), gomock.Any()).Return(nil)

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

// Test_Run_BindsTopicExchangeBeforeConsuming verifies the VOIP-1406 migration
// sequence inside Run(): QueueCreate -> every fanout QueueSubscribe ->
// TopicCreateWithKind(bin-manager.event) -> QueueBind(every topicPatterns entry)
// -> QueueUnbind(every fanoutUnbindTargets entry), all strictly BEFORE the
// ConsumeMessage goroutine is observed. The bind/unbind RPCs and ConsumeMessage's
// internal basic.consume share the queue's AMQP channel; racing them closes the
// channel with a 503 and the pod silently never consumes (VOIP-1258 PR #1101
// production incident, 2026-07-14).
func Test_Run_BindsTopicExchangeBeforeConsuming(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)

	queueName := string(commonoutline.QueueNameConversationSubscribe)
	subscribeTargets := []string{
		string(commonoutline.QueueNameMessageEvent),
		string(commonoutline.QueueNameEmailEvent),
		string(commonoutline.QueueNameWebchatEvent),
	}

	// callOrderCh, not a shared slice: QueueBind/QueueUnbind run synchronously on
	// the caller's goroutine, but ConsumeMessage's callback runs on Run()'s
	// internal goroutine -- appending to a plain slice from both would be a data
	// race. Buffered channel idiom follows bin-agent-manager's
	// Test_Run_BindsTopicExchangeBeforeReturning.
	callOrderCh := make(chan string, 16)

	// number of synchronous broker operations that must ALL be recorded before
	// ConsumeMessage may fire: QueueCreate + 3 fanout QueueSubscribe +
	// TopicCreateWithKind + len(topicPatterns) QueueBinds +
	// len(fanoutUnbindTargets) QueueUnbinds.
	syncOps := 1 + len(subscribeTargets) + 1 + len(topicPatterns) + len(fanoutUnbindTargets)

	calls := []any{
		mockSock.EXPECT().QueueCreate(queueName, "normal").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "QueueCreate"
				return nil
			}),
	}
	for _, target := range subscribeTargets {
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(queueName, target).
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "QueueSubscribe"
				return nil
			}))
	}

	// the VOIP-1406 bin-manager.event block: declare, bind every pattern, unbind
	// every fanout target -- strictly before ConsumeMessage.
	calls = append(calls,
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").
			DoAndReturn(func(_, _ string) error {
				callOrderCh <- "TopicCreateWithKind"
				return nil
			}),
	)
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(queueName, pattern, string(commonoutline.QueueNameEvent), false, nil).
			DoAndReturn(func(_, _, _ string, _ bool, _ interface{}) error {
				callOrderCh <- "QueueBind:topic"
				return nil
			}))
	}
	for _, target := range fanoutUnbindTargets {
		calls = append(calls, mockSock.EXPECT().QueueUnbind(queueName, "", target, nil).
			DoAndReturn(func(_, _, _ string, _ interface{}) error {
				callOrderCh <- "QueueUnbind:fanout"
				return nil
			}))
	}
	gomock.InOrder(calls...)

	// ConsumeMessage is started in a goroutine inside Run() -- allow it to be
	// called (or not, if the goroutine hasn't scheduled yet by the time Run()
	// returns) without blocking the test; the ordering assertion below is what
	// actually matters, not whether this fires.
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queueName, "conversation-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_, _, _ interface{}, _, _, _ bool, _ int, _ interface{}) error {
			callOrderCh <- "ConsumeMessage"
			return nil
		}).AnyTimes()

	h := NewSubscribeHandler(mockSock, queueName, subscribeTargets, mockAccount, mockConversation)

	if err := h.Run(); err != nil {
		t.Fatalf("Run() returned an unexpected error: %v", err)
	}

	// By the time Run() returns, every synchronous broker operation must already
	// have been recorded (they are called before the ConsumeMessage goroutine is
	// launched). Drain whatever has arrived so far without closing the channel --
	// ConsumeMessage's mock may still be in flight on its own goroutine and could
	// send after we're done reading.
	var callOrder []string
	draining := true
	for draining {
		select {
		case c := <-callOrderCh:
			callOrder = append(callOrder, c)
		default:
			draining = false
		}
	}

	seenSync := 0
	for _, c := range callOrder {
		// If ConsumeMessage's goroutine happened to race ahead and got recorded
		// before the full synchronous chain completed, the ordering bug is back.
		if c == "ConsumeMessage" {
			if seenSync < syncOps {
				t.Fatalf("ConsumeMessage was observed before the synchronous bind/unbind chain completed (%d/%d) -- ordering regression. callOrder: %v", seenSync, syncOps, callOrder)
			}
			continue
		}
		seenSync++
	}
	if seenSync != syncOps {
		t.Errorf("Expected %d synchronous broker operations to have been recorded within Run(), got %d. callOrder: %v", syncOps, seenSync, callOrder)
	}
}
