package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/pkg/emailhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

// closedLoopbackAddr returns a loopback address that nothing listens on.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

// handlerPattern returns the pattern the default mux has registered for the path, if any.
func handlerPattern(path string) string {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	_, pattern := http.DefaultServeMux.Handler(req)

	return pattern
}

func Test_initProm_disabled(t *testing.T) {
	tests := []struct {
		name string

		endpoint string
		listen   string
	}{
		{name: "empty_listen", endpoint: "/voip1652_disabled_a", listen: ""},
		{name: "empty_endpoint", endpoint: "", listen: "127.0.0.1:0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initProm(tt.endpoint, tt.listen)

			if tt.endpoint != "" {
				if pattern := handlerPattern(tt.endpoint); pattern != "" {
					t.Errorf("Expected no registered pattern for a disabled server. got: %q", pattern)
				}
			}
		})
	}
}

// promRuns makes the endpoint pattern unique per run: a pattern registers on the default mux
// once per process and a duplicate panics, so a rerun with -count must not reuse one.
var promRuns atomic.Int64

func Test_initProm_enabled(t *testing.T) {
	endpoint := fmt.Sprintf("/voip1652_metrics_%d", promRuns.Add(1))

	addr := closedLoopbackAddr(t)
	initProm(endpoint, addr)

	if pattern := handlerPattern(endpoint); pattern != endpoint {
		t.Fatalf("Wrong pattern. expect: %s, got: %q", endpoint, pattern)
	}

	// the metrics server answers from a goroutine, so poll until it is up
	var body string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + endpoint)
		if err == nil {
			data, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				body = string(data)
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	if !strings.Contains(body, "# HELP") {
		t.Errorf("Expected prometheus metrics text. got: %q", body)
	}
}

func Test_initProm_listenError(t *testing.T) {
	endpoint := fmt.Sprintf("/voip1652_occupied_%d", promRuns.Add(1))

	// hold the port so that the metrics server cannot bind it
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	hook := logrustest.NewGlobal()
	oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.StandardLogger().AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })

	initProm(endpoint, l.Addr().String())

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, "Prometheus server error") {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("Expected the listen error to be logged")
}

func Test_signalHandler(t *testing.T) {
	go signalHandler()

	chSigs <- syscall.SIGTERM

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Expected true on the done channel")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("The done channel was not signalled")
	}
}

func Test_createDBHandler_connectFails(t *testing.T) {
	// the address is refused, so the connection fails before any migration or cache is touched
	cfg := &config.Config{DatabaseDSN: fmt.Sprintf("user@tcp(%s)/db", closedLoopbackAddr(t))}

	res, err := createDBHandler(cfg)
	if err == nil || !strings.Contains(err.Error(), "database ping error") {
		t.Errorf("Wrong match. expect a ping error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil handler. got: %v", res)
	}
}

func Test_runListen(t *testing.T) {
	tests := []struct {
		name string

		queueErr error
	}{
		{name: "queue_create_fails_is_not_returned", queueErr: fmt.Errorf("queue failed")},
		{name: "listen_started", queueErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)

			consumed := make(chan struct{})
			mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(tt.queueErr)
			if tt.queueErr == nil {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ any) error {
						close(consumed)
						return nil
					})
			}

			// runListen logs the listen error and returns nil in both cases
			if err := runListen(mockSock, mockEmail); err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}

			if tt.queueErr == nil {
				select {
				case <-consumed:
				case <-time.After(2 * time.Second):
					t.Fatalf("ConsumeRPC was not called")
				}
			}
		})
	}
}
