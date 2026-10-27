package main

import (
	"context"
	"errors"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-number-manager/pkg/numberhandler"
	"monorepo/bin-number-manager/pkg/providerhandler"
)

const waitTimeout = 5 * time.Second

func Test_initSignal(t *testing.T) {
	// start from empty channels so a value left by another test cannot satisfy the assertions
	for len(chDone) > 0 {
		<-chDone
	}
	for len(chSigs) > 0 {
		<-chSigs
	}
	t.Cleanup(func() { signal.Stop(chSigs) })

	initSignal()

	// the signal is injected through the channel: no real signal is sent to the test process
	chSigs <- syscall.SIGTERM

	select {
	case got := <-chDone:
		if !got {
			t.Errorf("Wrong match. expect: true, got: %v", got)
		}
	case <-time.After(waitTimeout):
		t.Fatalf("timed out waiting for chDone")
	}
}

func Test_runServiceListen(t *testing.T) {
	tests := []struct {
		name string

		queueCreateErr error

		expectErr string
	}{
		{
			name: "normal",
		},
		{
			name:           "queue create error",
			queueCreateErr: errors.New("queue create failed"),
			expectErr:      "could not run the listenhandler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)
			mockProvider := providerhandler.NewMockProviderHandler(mc)

			queue := string(commonoutline.QueueNameNumberRequest)
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueCreateErr)

			consumed := make(chan struct{})
			if tt.queueCreateErr == nil {
				// Run starts the consumer in a goroutine: wait until it has called the mock
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "number-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, workerNum int, cbConsume sock.CbMsgRPC) error {
						close(consumed)
						return nil
					}).Times(1)
			}

			err := runServiceListen(mockSock, mockNumber, mockProvider)
			if tt.expectErr == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				select {
				case <-consumed:
				case <-time.After(waitTimeout):
					t.Fatalf("timed out waiting for the consumer")
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_runServiceSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameNumberSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		queueCreateErr error
		topicErr       error
		bindErr        error

		expectErr string
	}{
		{
			name: "normal",
		},
		{
			name:           "queue create error",
			queueCreateErr: errors.New("queue create failed"),
			expectErr:      "queue create failed",
		},
		{
			name:      "topic create error",
			topicErr:  errors.New("topic create failed"),
			expectErr: "topic create failed",
		},
		{
			name:      "queue bind error",
			bindErr:   errors.New("queue bind failed"),
			expectErr: "queue bind failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockNumber := numberhandler.NewMockNumberHandler(mc)

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueCreateErr)

			consumed := make(chan struct{})
			switch {
			case tt.queueCreateErr != nil:
			case tt.topicErr != nil:
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(tt.topicErr)
			case tt.bindErr != nil:
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil)
				mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(tt.bindErr).Times(1)
			default:
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil)
				// the pattern strings are checked by the golden test of the subscribe handler
				mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(nil).Times(2)
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "number-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(ctx context.Context, queueName string, consumerName string, exclusive bool, noLocal bool, noWait bool, numWorkers int, messageConsume sock.CbMsgConsume) error {
						close(consumed)
						return nil
					}).Times(1)
			}

			err := runServiceSubscribe(mockSock, mockNumber)
			if tt.expectErr == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				select {
				case <-consumed:
				case <-time.After(waitTimeout):
					t.Fatalf("timed out waiting for the consumer")
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}
