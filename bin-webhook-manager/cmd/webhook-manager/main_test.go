package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/activeflowhandler"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

func Test_runListen(t *testing.T) {
	queue := string(commonoutline.QueueNameWebhookRequest)

	tests := []struct {
		name string

		queueCreateErr error

		expectConsume     bool
		expectErrContains string
	}{
		{
			name:              "queue create failure is wrapped and nothing is consumed",
			queueCreateErr:    errors.New("broker down"),
			expectErrContains: "could not run the listen handler correctly",
		},
		{
			name:          "success consumes the webhook request queue",
			expectConsume: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockActiveflow := activeflowhandler.NewMockActiveflowHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueCreateErr)

			consumed := make(chan struct{})
			if tt.expectConsume {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, "webhook-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ interface{}, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
						close(consumed)
						return nil
					})
			}

			err := runListen(mockSock, mockReq, mockNotify, mockAccount, mockActiveflow, mockDB)
			if tt.expectErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrContains, err)
				}
				mc.Finish()
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}

			// the consumer runs in a goroutine; wait for it before the controller checks the calls.
			select {
			case <-consumed:
			case <-time.After(5 * time.Second):
				t.Fatalf("ConsumeRPC was not called")
			}
			mc.Finish()
		})
	}
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameWebhookSubscribe)
	eventExchange := string(commonoutline.QueueNameEvent)

	tests := []struct {
		name string

		queueCreateErr error
		topicErr       error

		expectBind        bool
		expectErrContains string
	}{
		{
			name:              "queue create failure is wrapped",
			queueCreateErr:    errors.New("broker down"),
			expectErrContains: "could not run the subscribe handler correctly",
		},
		{
			name:              "topic exchange failure is wrapped",
			topicErr:          errors.New("exchange down"),
			expectErrContains: "could not run the subscribe handler correctly",
		},
		{
			name:       "success binds the event exchange and consumes the subscribe queue",
			expectBind: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueCreateErr)
			if tt.queueCreateErr == nil {
				mockSock.EXPECT().TopicCreateWithKind(eventExchange, "topic").Return(tt.topicErr)
			}

			consumed := make(chan struct{})
			if tt.expectBind {
				mockSock.EXPECT().QueueBind(queue, gomock.Any(), eventExchange, false, nil).Return(nil).Times(5)
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, "webhook-manager", false, false, false, 10, gomock.Any()).DoAndReturn(
					func(_ interface{}, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
						close(consumed)
						return nil
					})
			}

			err := runSubscribe(mockSock, mockAccount, mockCache)
			if tt.expectErrContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrContains) {
					t.Errorf("Wrong match. expect error containing %q, got: %v", tt.expectErrContains, err)
				}
				mc.Finish()
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(5 * time.Second):
				t.Fatalf("ConsumeMessage was not called")
			}
			mc.Finish()
		})
	}
}

func Test_initSignal(t *testing.T) {
	// drain a left-over done signal so that only this test's signal is observed.
	select {
	case <-chDone:
	default:
	}

	initSignal()

	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		signal.Stop(chSigs)
		t.Fatalf("Could not send the signal. err: %v", err)
	}

	// the handler must still be registered while the signal is delivered, so stop it only after
	// the done signal has been seen.
	select {
	case <-chDone:
	case <-time.After(5 * time.Second):
		signal.Stop(chSigs)
		t.Fatalf("The done signal was not sent")
	}
	signal.Stop(chSigs)
}

// logCollector is a logrus hook that records messages and notifies a waiter per message.
type logCollector struct {
	mu   sync.Mutex
	msgs []string
	ch   chan string
}

func (c *logCollector) Levels() []logrus.Level { return logrus.AllLevels }

func (c *logCollector) Fire(e *logrus.Entry) error {
	c.mu.Lock()
	c.msgs = append(c.msgs, e.Message)
	c.mu.Unlock()

	select {
	case c.ch <- e.Message:
	default:
	}
	return nil
}

func (c *logCollector) contains(sub string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range c.msgs {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

var (
	initPromOnce sync.Once
	promLogs     *logCollector
)

// runInitPromOnce calls initProm with the cases of Test_initProm exactly once per process, because
// a registered pattern of http.DefaultServeMux cannot be removed and a repeated registration panics.
func runInitPromOnce(t *testing.T) {
	t.Helper()

	initPromOnce.Do(func() {
		promLogs = &logCollector{ch: make(chan string, 64)}

		origLevel := logrus.GetLevel()
		origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
		logrus.AddHook(promLogs)
		logrus.SetLevel(logrus.DebugLevel)
		defer func() {
			logrus.SetLevel(origLevel)
			logrus.StandardLogger().ReplaceHooks(origHooks)
		}()

		// disabled: either value empty
		initProm("", "127.0.0.1:0")
		initProm("/prom-disabled-b", "")

		// enabled: an invalid port makes the listener fail at once and the failure is only logged.
		initProm("/prom-enabled", "127.0.0.1:99999")

		timeout := time.After(5 * time.Second)
		for !promLogs.contains("Prometheus server error") {
			select {
			case <-promLogs.ch:
			case <-timeout:
				return
			}
		}
	})
}

func Test_initProm(t *testing.T) {
	runInitPromOnce(t)

	t.Run("disabled configuration only logs", func(t *testing.T) {
		if !promLogs.contains("Prometheus metrics disabled") {
			t.Errorf("Wrong match. expect: disabled message logged")
		}

		req := httptest.NewRequest(http.MethodGet, "/prom-disabled-b", nil)
		if _, pattern := http.DefaultServeMux.Handler(req); pattern != "" {
			t.Errorf("Wrong match. expect: no registered pattern, got: %q", pattern)
		}
	})

	t.Run("enabled configuration registers the metrics handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/prom-enabled", nil)
		w := httptest.NewRecorder()
		http.DefaultServeMux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
		}
		if !strings.Contains(w.Body.String(), "# HELP") {
			t.Errorf("Wrong match. expect: prometheus exposition text, got: %q", w.Body.String())
		}
	})

	t.Run("listener failure is logged and not fatal", func(t *testing.T) {
		if !promLogs.contains("Prometheus metrics server starting on 127.0.0.1:99999/prom-enabled") {
			t.Errorf("Wrong match. expect: start message logged")
		}
		if !promLogs.contains("Prometheus server error") {
			t.Errorf("Wrong match. expect: server error logged")
		}
	})
}
