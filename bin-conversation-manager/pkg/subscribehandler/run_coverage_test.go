package subscribehandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/internal/logtest"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

var errBrokerCoverage = errors.New("broker failure")

func Test_Run_failures(t *testing.T) {
	queue := string(commonoutline.QueueNameConversationSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		queueErr  error
		topicErr  error
		bindErrAt int // 1-based index of the bind that fails, 0 for none

		expectErr string
	}{
		{name: "queue creation failure", queueErr: errBrokerCoverage, expectErr: "could not declare the queue for subscribeHandler"},
		{name: "topic exchange failure", topicErr: errBrokerCoverage, expectErr: "could not declare the global topic exchange"},
		{name: "first bind failure", bindErrAt: 1, expectErr: "could not bind the topic pattern. pattern: " + topicPatterns[0]},
		{name: "last bind failure", bindErrAt: len(topicPatterns), expectErr: "could not bind the topic pattern. pattern: " + topicPatterns[len(topicPatterns)-1]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mock is strict: a call after the failing step is not declared and fails the test, which also
			// proves that no consumer is started after a failure
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr).Times(1)
			if tt.queueErr == nil {
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(tt.topicErr).Times(1)
			}
			if tt.queueErr == nil && tt.topicErr == nil {
				for i, pattern := range topicPatterns {
					var bindErr error
					if i+1 == tt.bindErrAt {
						bindErr = errBrokerCoverage
					}
					mockSock.EXPECT().QueueBind(queue, pattern, exchange, false, nil).Return(bindErr).Times(1)
					if bindErr != nil {
						break
					}
				}
			}

			h := NewSubscribeHandler(mockSock, queue, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc))

			err := h.Run()
			if !errors.Is(err, errBrokerCoverage) {
				t.Errorf("Run() error = %v, want %v wrapped", err, errBrokerCoverage)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Run() error = %v, want it to contain %q", err, tt.expectErr)
			}
		})
	}
}

// consumeSignal is a logrus hook that signals when an entry with the given text is logged.
type consumeSignal struct {
	text string
	seen chan struct{}
}

func (s *consumeSignal) Levels() []logrus.Level { return logrus.AllLevels }

func (s *consumeSignal) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, s.text) {
		select {
		case s.seen <- struct{}{}:
		default:
		}
	}

	return nil
}

func Test_Run_consumerFailureIsLogged(t *testing.T) {
	queue := string(commonoutline.QueueNameConversationSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	mc := gomock.NewController(t)
	defer mc.Finish()

	// the failure is logged by a goroutine, so the log is waited on instead of a fixed delay
	logtest.Capture(t)
	logged := &consumeSignal{text: "Could not consume the request message correctly", seen: make(chan struct{}, 1)}
	logrus.AddHook(logged)

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
	mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil)
	mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(nil).Times(len(topicPatterns))
	consuming := make(chan struct{})
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "conversation-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ commonsock.CbMsgConsume) error {
			close(consuming)
			return errBrokerCoverage
		}).Times(1)

	h := NewSubscribeHandler(mockSock, queue, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc))
	if err := h.Run(); err != nil {
		t.Fatalf("Run() error = %v, want nil: a consumer failure is only logged", err)
	}

	select {
	case <-consuming:
	case <-time.After(5 * time.Second):
		t.Fatalf("the consumer was not started")
	}
	select {
	case <-logged.seen:
	case <-time.After(5 * time.Second):
		t.Fatalf("the consumer failure was not logged")
	}
}

func Test_processEventRun(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockConversation := conversationhandler.NewMockConversationHandler(mc)
	h := &subscribeHandler{
		sockHandler:         sockhandler.NewMockSockHandler(mc),
		accountHandler:      accounthandler.NewMockAccountHandler(mc),
		conversationHandler: mockConversation,
	}

	data, err := json.Marshal(&mmmessage.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2d7c9e10-5a3b-4f6e-8b1c-0a9d8e7f6a50"),
			CustomerID: uuid.FromStringOrNil("2d7c9e10-5a3b-4f6e-8b1c-0a9d8e7f6a51"),
		},
		Text: "test message",
	})
	if err != nil {
		t.Fatalf("could not build the event data: %v", err)
	}

	event := &commonsock.Event{
		Publisher: publisherMessageManager,
		Type:      string(mmmessage.EventTypeMessageCreated),
		Data:      json.RawMessage(data),
	}

	// the event is processed by a goroutine, so the handler call is waited on
	handled := make(chan struct{})
	mockConversation.EXPECT().Event(gomock.Any(), conversation.TypeMessage, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ conversation.Type, _ any) error {
			close(handled)
			return nil
		}).Times(1)

	if err := h.processEventRun(event); err != nil {
		t.Errorf("processEventRun() error = %v, want nil", err)
	}

	select {
	case <-handled:
	case <-time.After(5 * time.Second):
		t.Fatalf("the event was not dispatched to the conversation handler")
	}
}
