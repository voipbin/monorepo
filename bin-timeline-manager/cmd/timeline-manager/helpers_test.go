package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-timeline-manager/internal/config"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// helperSetConfig changes the process-wide config for one test and restores it afterwards.
func helperSetConfig(t *testing.T, mutate func(c *config.Config)) {
	t.Helper()

	c := config.Get()
	orig := *c
	mutate(c)
	t.Cleanup(func() { *c = orig })
}

// helperDrainSignalChannels empties the package signal channels so a value left by an earlier test
// cannot satisfy an assertion.
func helperDrainSignalChannels() {
	for {
		select {
		case <-chSigs:
		case <-chDone:
		default:
			return
		}
	}
}

func Test_runMigrations(t *testing.T) {
	tests := []struct {
		name string

		address string

		expectErr string
	}{
		{
			name:    "empty address skips the migrations",
			address: "",
		},
		{
			name:    "address that nothing listens on returns the initialization error",
			address: "127.0.0.1:1",

			expectErr: "failed to initialize migrate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			helperSetConfig(t, func(c *config.Config) {
				c.ClickHouseAddress = tt.address
				c.ClickHouseDatabase = "test_db"
				c.MigrationsPath = dir
			})

			err := runMigrations()
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_initSignal(t *testing.T) {
	helperDrainSignalChannels()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		helperDrainSignalChannels()
	})

	initSignal()
	chSigs <- syscall.SIGTERM

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Wrong match. expect: true, got: false")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect a done notification after the signal, got: none")
	}
}

func Test_initProm_disabled(t *testing.T) {
	tests := []struct {
		name string

		endpoint string
		listen   string
	}{
		{name: "both empty", endpoint: "", listen: ""},
		{name: "endpoint without listen address", endpoint: "/test-metrics-a", listen: ""},
		{name: "listen address without endpoint", endpoint: "", listen: "127.0.0.1:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initProm(tt.endpoint, tt.listen)

			// nothing may be registered on the default mux for the endpoint
			if tt.endpoint != "" {
				_, pattern := http.DefaultServeMux.Handler(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: tt.endpoint}})
				if pattern != "" {
					t.Errorf("Wrong match. expect no handler for %s, got pattern: %s", tt.endpoint, pattern)
				}
			}
		})
	}
}

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameTimelineRequest)

	t.Run("queue creation failure is logged and nil is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(fmt.Errorf("simulated queue failure"))

		// the queue failure is logged and no error is returned to the caller
		if err := runListen(mockSock, nil, nil, nil, nil); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	})

	t.Run("success starts the consumer on the request queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		consumed := make(chan string, 1)

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "timeline-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
			func(_ context.Context, queueName, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
				consumed <- queueName
				return nil
			},
		)

		if err := runListen(mockSock, nil, nil, nil, nil); err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}

		select {
		case got := <-consumed:
			if got != queue {
				t.Errorf("Wrong match. expect: %s, got: %s", queue, got)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect the consumer to be started, got: none")
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	t.Run("queue creation failure returns the error and no channel", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameTimelineSubscribe), "normal").Return(fmt.Errorf("simulated queue failure"))

		done, err := runSubscribe(context.Background(), mockSock, mockDB)
		if err == nil || !strings.Contains(err.Error(), "could not declare the queue") {
			t.Errorf("Wrong match. expect error containing: could not declare the queue, got: %v", err)
		}
		if done != nil {
			t.Errorf("Wrong match. expect: nil channel, got: non-nil")
		}
	})

	t.Run("success returns the channel that closes after the context is cancelled", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		consumeStarted := make(chan struct{}, 1)

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockDB := dbhandler.NewMockDBHandler(mc)
		mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameTimelineSubscribe), "normal").Return(nil)
		mockSock.EXPECT().QueueSubscribe(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSock.EXPECT().QueueUnbind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(commonoutline.QueueNameTimelineSubscribe), "timeline-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
			func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
				consumeStarted <- struct{}{}
				return nil
			},
		)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done, err := runSubscribe(ctx, mockSock, mockDB)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if done == nil {
			t.Fatalf("Wrong match. expect: a channel, got: nil")
		}

		select {
		case <-consumeStarted:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect the consumer to be started, got: none")
		}

		select {
		case <-done:
			t.Fatalf("Wrong match. expect the channel to stay open while the context is active")
		default:
		}

		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect the channel to close after the cancel, got: still open")
		}
	})
}

func Test_runDaemon_migration_failure(t *testing.T) {
	helperDrainSignalChannels()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		helperDrainSignalChannels()
	})

	dir := t.TempDir()
	helperSetConfig(t, func(c *config.Config) {
		c.PrometheusEndpoint = ""
		c.PrometheusListenAddress = ""
		c.ClickHouseAddress = "127.0.0.1:1"
		c.ClickHouseDatabase = "test_db"
		c.MigrationsPath = dir
	})

	// runDaemon waits on the done channel only after the migrations succeed, so a regression that
	// ignores the migration failure must fail the test instead of blocking it
	errCh := make(chan error, 1)
	go func() { errCh <- runDaemon() }()

	var err error
	select {
	case err = <-errCh:
	case <-time.After(10 * time.Second):
		t.Fatalf("Wrong match. expect runDaemon to return the migration error, got: still running")
	}
	if err == nil || !strings.Contains(err.Error(), "could not run migrations") {
		t.Errorf("Wrong match. expect error containing: could not run migrations, got: %v", err)
	}
}
