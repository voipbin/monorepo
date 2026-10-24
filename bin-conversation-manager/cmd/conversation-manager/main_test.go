package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

var errBrokerTest = errors.New("broker failure")

func Test_runListen(t *testing.T) {
	const queue = "bin-manager.conversation-manager.request"

	t.Run("creates the request queue and starts the rpc consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consuming := make(chan struct{})
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil).Times(1)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "conversation-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
				close(consuming)
				return nil
			}).Times(1)

		err := runListen(mockSock, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc), messagehandler.NewMockMessageHandler(mc))
		if err != nil {
			t.Fatalf("runListen() error = %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("the rpc consumer was not started")
		}
	})

	// characterization of the current behavior: the queue failure is returned wrapped. If the
	// handling changes, update this case.
	t.Run("a queue failure is returned and no consumer is started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errBrokerTest).Times(1)

		err := runListen(mockSock, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc), messagehandler.NewMockMessageHandler(mc))
		if !errors.Is(err, errBrokerTest) {
			t.Errorf("runListen() error = %v, want %v wrapped", err, errBrokerTest)
		}
		if err == nil || !strings.Contains(err.Error(), "could not declare the queue for listenHandler") {
			t.Errorf("runListen() error = %v, want the queue context", err)
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	const (
		queue    = "bin-manager.conversation-manager.subscribe"
		exchange = "bin-manager.event"
	)

	// the bind patterns are pinned by the golden test of the subscribehandler package. Update
	// this list together with it.
	patterns := []string{
		"message-manager.message.*.created",
		"email-manager.email.*.created",
		"email-manager.email.*.updated",
		"webchat-manager.webchat.*.message_created",
	}

	tests := []struct {
		name string

		queueErr  error
		topicErr  error
		bindErrAt int // 1-based index of the bind that fails, 0 for none

		expectErr string
	}{
		{name: "queue creation failure", queueErr: errBrokerTest, expectErr: "could not declare the queue for subscribeHandler"},
		{name: "topic exchange failure", topicErr: errBrokerTest, expectErr: "could not declare the global topic exchange"},
		{name: "first bind failure", bindErrAt: 1, expectErr: "could not bind the topic pattern. pattern: " + patterns[0]},
		{name: "last bind failure", bindErrAt: len(patterns), expectErr: "could not bind the topic pattern. pattern: " + patterns[len(patterns)-1]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the mock is strict: calls after the failing step are not expected and fail the test
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr).Times(1)
			if tt.queueErr == nil {
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(tt.topicErr).Times(1)
			}
			if tt.queueErr == nil && tt.topicErr == nil {
				n := len(patterns)
				if tt.bindErrAt > 0 {
					n = tt.bindErrAt
				}
				for i := 0; i < n; i++ {
					var ret error
					if i+1 == tt.bindErrAt {
						ret = errBrokerTest
					}
					mockSock.EXPECT().QueueBind(queue, patterns[i], exchange, false, nil).Return(ret).Times(1)
				}
			}

			err := runSubscribe(mockSock, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc))
			if !errors.Is(err, errBrokerTest) {
				t.Errorf("runSubscribe() error = %v, want %v wrapped", err, errBrokerTest)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("runSubscribe() error = %v, want it to contain %q", err, tt.expectErr)
			}
		})
	}

	t.Run("binds every pattern in order and starts the consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consuming := make(chan struct{})

		calls := []any{
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil).Times(1),
			mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil).Times(1),
		}
		for _, p := range patterns {
			calls = append(calls, mockSock.EXPECT().QueueBind(queue, p, exchange, false, nil).Return(nil).Times(1))
		}
		gomock.InOrder(calls...)

		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "conversation-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
				close(consuming)
				return nil
			}).Times(1)

		err := runSubscribe(mockSock, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc))
		if err != nil {
			t.Fatalf("runSubscribe() error = %v", err)
		}

		select {
		case <-consuming:
		case <-time.After(5 * time.Second):
			t.Fatalf("the event consumer was not started")
		}
	})
}
