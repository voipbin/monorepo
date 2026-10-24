package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-tag-manager/pkg/taghandler"
)

func Test_rootCmd_flag_defaults(t *testing.T) {
	// the package init() has registered the flags on rootCmd
	stringFlags := map[string]string{
		"database_dsn":              "testid:testpassword@tcp(127.0.0.1:3306)/test",
		"prometheus_endpoint":       "/metrics",
		"prometheus_listen_address": ":2112",
		"rabbitmq_address":          "amqp://guest:" + "guest" + "@localhost:5672",
		"redis_address":             "127.0.0.1:6379",
		"redis_password":            "",
	}
	for name, expect := range stringFlags {
		f := rootCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag %s is not registered", name)
			continue
		}
		if f.Value.Type() != "string" {
			t.Errorf("flag %s must be a string flag, got: %s", name, f.Value.Type())
		}
		if f.DefValue != expect {
			t.Errorf("Wrong default for %s. expect: %q, got: %q", name, expect, f.DefValue)
		}
	}

	db := rootCmd.Flags().Lookup("redis_database")
	if db == nil {
		t.Fatalf("flag redis_database is not registered")
	}
	if db.Value.Type() != "int" || db.DefValue != "1" {
		t.Errorf("Wrong redis_database flag. expect: int 1, got: %s %s", db.Value.Type(), db.DefValue)
	}
}

func Test_signalHandler(t *testing.T) {
	// The init() of this package started a goroutine running signalHandler. It consumes exactly one
	// value from chSigs, so it is restarted in the cleanup to keep -count=N runs working. The test
	// never sends a real OS signal and never calls signalHandler directly.
	// Note: signal.Notify in init() intercepts SIGINT, SIGTERM and SIGHUP for the test binary, so a
	// run that fails half way can leave a second signalHandler goroutine behind.
	t.Cleanup(func() { go signalHandler() })

	chSigs <- nil

	select {
	case v := <-chDone:
		if !v {
			t.Errorf("chDone must receive true")
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("signalHandler did not report done")
	}
}

func Test_runSubscribe(t *testing.T) {
	const (
		subscribeQueue = "bin-manager.tag-manager.subscribe"
		eventExchange  = "bin-manager.event"
	)

	tests := []struct {
		name string

		queueCreateErr error
		topicErr       error
		bindErr        error

		expectErr    string
		expectBinds  int
		expectTopic  bool
		expectConsum bool
	}{
		{
			name:           "queue create fails",
			queueCreateErr: errors.New("queue failure"),
			expectErr:      "could not declare the queue for subscribeHandler",
		},
		{
			name:        "topic exchange declaration fails",
			topicErr:    errors.New("topic failure"),
			expectErr:   "could not declare the global topic exchange",
			expectTopic: true,
		},
		{
			name:        "queue bind fails",
			bindErr:     errors.New("bind failure"),
			expectErr:   "could not bind the topic pattern",
			expectTopic: true,
			expectBinds: 1,
		},
		{
			name:         "success",
			expectTopic:  true,
			expectBinds:  1,
			expectConsum: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTag := taghandler.NewMockTagHandler(mc)

			mockSock.EXPECT().QueueCreate(subscribeQueue, "normal").Return(tt.queueCreateErr)
			if tt.expectTopic {
				mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(tt.topicErr)
			}

			binds := 0
			if tt.expectBinds > 0 {
				mockSock.EXPECT().QueueBind(subscribeQueue, gomock.Any(), eventExchange, false, gomock.Nil()).
					DoAndReturn(func(string, string, string, bool, any) error {
						binds++
						return tt.bindErr
					}).Times(tt.expectBinds)
			}

			consumed := make(chan struct{})
			if tt.expectConsum {
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), subscribeQueue, "tag-manager", false, false, false, 10, gomock.Any()).
					DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgConsume) error {
						close(consumed)
						return nil
					})
			}

			err := runSubscribe(mockSock, mockTag)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong error. expect: %s, got: %v", tt.expectErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(3 * time.Second):
				t.Fatalf("ConsumeMessage was not started")
			}
			if binds != tt.expectBinds {
				t.Errorf("Wrong bind count. expect: %d, got: %d", tt.expectBinds, binds)
			}
		})
	}
}

func Test_runListen(t *testing.T) {
	const requestQueue = "bin-manager.tag-manager.request"

	t.Run("queue create failure is logged and nil is returned", func(t *testing.T) {
		// Current behavior: runListen logs the error from listenHandler.Run and returns nil.
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTag := taghandler.NewMockTagHandler(mc)
		mockSock.EXPECT().QueueCreate(requestQueue, "normal").Return(errors.New("queue failure"))

		if err := runListen(mockSock, mockTag); err != nil {
			t.Errorf("runListen must return nil on a QueueCreate failure, got: %v", err)
		}
	})

	t.Run("success starts the request consumer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTag := taghandler.NewMockTagHandler(mc)

		consumed := make(chan struct{})
		mockSock.EXPECT().QueueCreate(requestQueue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), requestQueue, "tag-manager", false, false, false, 10, gomock.Any()).
			DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
				close(consumed)
				return nil
			})

		if err := runListen(mockSock, mockTag); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		select {
		case <-consumed:
		case <-time.After(3 * time.Second):
			t.Fatalf("ConsumeRPC was not started")
		}
	})
}
