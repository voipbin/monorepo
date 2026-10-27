package main

import (
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

const consumeWait = 3 * time.Second

var errSockBoom = errors.New("injected sock failure")

func Test_init_flags(t *testing.T) {
	if rootCmd.Use != "contact-manager" {
		t.Errorf("Wrong match. expect: contact-manager, got: %s", rootCmd.Use)
	}
	if rootCmd.RunE == nil {
		t.Errorf("Wrong match. expect: RunE set, got: nil")
	}

	tests := []struct {
		name string

		expectType    string
		expectDefault string
		checkDefault  bool
	}{
		{name: "database_dsn", expectType: "string"},
		{name: "prometheus_endpoint", expectType: "string", expectDefault: "/metrics", checkDefault: true},
		{name: "prometheus_listen_address", expectType: "string", expectDefault: ":2112", checkDefault: true},
		{name: "rabbitmq_address", expectType: "string"},
		{name: "redis_address", expectType: "string"},
		{name: "redis_database", expectType: "int", expectDefault: "1", checkDefault: true},
		{name: "redis_password", expectType: "string", expectDefault: "", checkDefault: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := rootCmd.Flags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("Wrong match. expect: flag %s, got: none", tt.name)
			}
			if f.Value.Type() != tt.expectType {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectType, f.Value.Type())
			}
			if tt.checkDefault && f.DefValue != tt.expectDefault {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectDefault, f.DefValue)
			}
		})
	}
}

var signalOnce sync.Once

// init() starts the only goroutine that reads chSigs, and it reads once, so the signal
// path can be exercised once per process; it is never started a second time here.
func Test_signalHandler(t *testing.T) {
	ran := false
	signalOnce.Do(func() {
		ran = true

		chSigs <- syscall.SIGTERM

		select {
		case done := <-chDone:
			if !done {
				t.Errorf("Wrong match. expect: true, got: false")
			}
		case <-time.After(consumeWait):
			t.Fatalf("Wrong match. expect: done signal, got: timeout")
		}
	})
	if !ran {
		t.Skip("the one-shot signal goroutine was already consumed by an earlier run")
	}

	if len(chDone) != 0 || len(chSigs) != 0 {
		t.Errorf("Wrong match. expect: drained channels, got: sigs %d, done %d", len(chSigs), len(chDone))
	}
}

func Test_runListen(t *testing.T) {
	tests := []struct {
		name string

		queueCreateErr error

		expectConsume bool
	}{
		{name: "consumer is started on the request queue", expectConsume: true},
		{name: "queue creation failure is logged and not returned", queueCreateErr: errSockBoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)
			mockAddress := addresshandler.NewMockAddressHandler(mc)
			mockCase := casehandler.NewMockCaseHandler(mc)

			consumed := make(chan string, 1)

			mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameContactRequest), "normal").Return(tt.queueCreateErr)
			if tt.expectConsume {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), string(commonoutline.QueueNameContactRequest), "contact-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, queue, _ string, _, _, _ bool, _ int, _ any) error {
						consumed <- queue
						return nil
					})
			}

			if err := runListen(mockSock, mockContact, mockAddress, mockCase); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !tt.expectConsume {
				return
			}
			select {
			case queue := <-consumed:
				if queue != string(commonoutline.QueueNameContactRequest) {
					t.Errorf("Wrong match. expect: %s, got: %s", commonoutline.QueueNameContactRequest, queue)
				}
			case <-time.After(consumeWait):
				t.Fatalf("Wrong match. expect: consumer started, got: timeout")
			}
		})
	}
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameContactSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		queueCreateErr error
		topicErr       error
		bindErr        error

		expectTopic   bool
		expectBind    bool
		expectConsume bool
	}{
		{name: "consumer is started", expectTopic: true, expectBind: true, expectConsume: true},
		{name: "queue creation fails", queueCreateErr: errSockBoom},
		{name: "topic exchange declaration fails", topicErr: errSockBoom, expectTopic: true},
		{name: "topic binding fails", bindErr: errSockBoom, expectTopic: true, expectBind: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)

			consumed := make(chan string, 1)

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueCreateErr)
			if tt.expectTopic {
				mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(tt.topicErr)
			}
			if tt.expectBind {
				mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(tt.bindErr).MinTimes(1)
			}
			if tt.expectConsume {
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "contact-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, q, _ string, _, _, _ bool, _ int, _ any) error {
						consumed <- q
						return nil
					})
			}

			err := runSubscribe(mockSock, mockContact)

			expectErr := tt.queueCreateErr != nil || tt.topicErr != nil || tt.bindErr != nil
			if expectErr {
				if !errors.Is(err, errSockBoom) {
					t.Errorf("Wrong match. expect: injected error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			select {
			case q := <-consumed:
				if q != queue {
					t.Errorf("Wrong match. expect: %s, got: %s", queue, q)
				}
			case <-time.After(consumeWait):
				t.Fatalf("Wrong match. expect: consumer started, got: timeout")
			}
		})
	}
}
