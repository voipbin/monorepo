package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/pkg/streaminghandler"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_runListen(t *testing.T) {
	hostID := uuid.FromStringOrNil("b25a1e10-0000-11ee-9c11-000000000001")

	tests := []struct {
		name string

		queueErr         error
		volatileQueueErr error

		expectConsumers int
		expectErr       string
	}{
		{
			name:            "both queues are declared and consumed",
			expectConsumers: 2,
		},
		{
			name:      "the normal queue declaration fails",
			queueErr:  fmt.Errorf("declare failed"),
			expectErr: "could not run the listenhandler correctly",
		},
		{
			name:             "the volatile queue declaration fails after the first consumer started",
			volatileQueueErr: fmt.Errorf("declare failed"),
			expectConsumers:  1,
			expectErr:        "could not run the listenhandler correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			consumed := make(chan string, 2)
			consume := func() {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ context.Context, name string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
						consumed <- name
						return nil
					})
			}

			mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(tt.queueErr)
			if tt.queueErr == nil {
				consume()
				mockSock.EXPECT().QueueCreate(fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID), "volatile").Return(tt.volatileQueueErr)
				if tt.volatileQueueErr == nil {
					consume()
				}
			}

			err := runListen(mockSock, hostID, mockReq, mockTranscript, mockTranscribe)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the consumers run in goroutines; wait for all of them before the controller is finished
			got := map[string]bool{}
			for i := 0; i < tt.expectConsumers; i++ {
				select {
				case name := <-consumed:
					got[name] = true
				case <-time.After(5 * time.Second):
					t.Fatalf("Wrong match. expect %d consumers, got: %v", tt.expectConsumers, got)
				}
			}
			if tt.expectConsumers == 2 && !got[fmt.Sprintf("bin-manager.transcribe-manager-%s.request", hostID)] {
				t.Errorf("Wrong match. expect the per-host queue to be consumed, got: %v", got)
			}
		})
	}
}

func Test_runSubscribe(t *testing.T) {
	tests := []struct {
		name string

		setup func(mockSock *sockhandler.MockSockHandler, consumed chan struct{})

		expectConsumer bool
		expectErr      string
	}{
		{
			name: "the queue declaration fails",
			setup: func(mockSock *sockhandler.MockSockHandler, _ chan struct{}) {
				mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(fmt.Errorf("declare failed"))
			},
			expectErr: "could not run the subscribehandler correctly",
		},
		{
			name: "the topic exchange declaration fails",
			setup: func(mockSock *sockhandler.MockSockHandler, _ chan struct{}) {
				mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil)
				mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(fmt.Errorf("declare failed"))
			},
			expectErr: "could not run the subscribehandler correctly",
		},
		{
			name: "the binding fails",
			setup: func(mockSock *sockhandler.MockSockHandler, _ chan struct{}) {
				mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil)
				mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(nil)
				mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), false, nil).Return(fmt.Errorf("bind failed"))
			},
			expectErr: "could not run the subscribehandler correctly",
		},
		{
			name: "everything is declared and the consumer starts",
			setup: func(mockSock *sockhandler.MockSockHandler, consumed chan struct{}) {
				mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil)
				mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(nil)
				// the list of binding patterns is internal to the subscribe handler
				mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), false, nil).Return(nil).AnyTimes()
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
						close(consumed)
						return nil
					})
			},
			expectConsumer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

			consumed := make(chan struct{})
			tt.setup(mockSock, consumed)

			err := runSubscribe(mockSock, mockTranscribe)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectConsumer {
				select {
				case <-consumed:
				case <-time.After(5 * time.Second):
					t.Fatalf("Wrong match. expect the consumer to start")
				}
			}
		})
	}
}

func Test_runStreaming(t *testing.T) {
	tests := []struct {
		name string

		runErr error

		expectErr string
	}{
		{
			name: "the streaming handler runs",
		},
		{
			name:      "the streaming handler fails to run",
			runErr:    fmt.Errorf("run failed"),
			expectErr: "could not run the streaminghandler correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)
			mockStreaming.EXPECT().Run().Return(tt.runErr)

			err := runStreaming(mockStreaming)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_runDaemon_databaseError checks that the daemon stops with the wrapped connect error when the
// database is not reachable, and that it does so before the cache, the migrations or the broker.
func Test_runDaemon_databaseError(t *testing.T) {
	saveConfig(t)
	config.Get().PrometheusEndpoint = ""
	config.Get().PrometheusListenAddress = ""
	config.Get().DatabaseDSN = "tester@tcp(" + closedAddr(t) + ")/transcribe_test"

	// runDaemon starts the signal goroutine; end it in cleanup so it does not leak into other tests.
	t.Cleanup(func() {
		chSigs <- os.Interrupt
		select {
		case <-chDone:
		case <-time.After(5 * time.Second):
			t.Error("the signal goroutine did not report done")
		}
		signal.Stop(chSigs)
	})

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expect: wrapped database error, got: %v", err)
	}
}
