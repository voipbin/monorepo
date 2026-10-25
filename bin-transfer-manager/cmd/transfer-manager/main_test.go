package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/internal/config"
	"monorepo/bin-transfer-manager/pkg/transferhandler"
)

const (
	testUnreachableDSN = "tester@unix(/nonexistent/transfer-manager-test.sock)/test"
	testWaitTimeout    = 5 * time.Second
)

var testEndpointSeq atomic.Int64

// newEndpoint returns a metrics path that was not registered on http.DefaultServeMux before.
// The registration cannot be undone, so every initProm call of the tests needs its own path.
func newEndpoint() string {
	return fmt.Sprintf("/test-metrics-%d", testEndpointSeq.Add(1))
}

// setConfig replaces the global config for a test and restores it afterwards.
func setConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	saved := *config.Get()
	t.Cleanup(func() { *config.Get() = saved })

	*config.Get() = cfg
}

// messageHook collects the messages of the log entries that contain the given text.
type messageHook struct {
	contains string
	ch       chan string
}

func (h *messageHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *messageHook) Fire(e *logrus.Entry) error {
	if !strings.Contains(e.Message, h.contains) {
		return nil
	}

	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

// hookLog registers a hook on the standard logger and removes it when the test ends.
func hookLog(t *testing.T, contains string) chan string {
	t.Helper()

	h := &messageHook{contains: contains, ch: make(chan string, 8)}
	saved := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(h)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(saved) })

	return h.ch
}

func Test_initSignal(t *testing.T) {
	// drop a leftover value of an earlier test
	select {
	case <-chDone:
	default:
	}

	initSignal()
	t.Cleanup(func() { signal.Stop(chSigs) })

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("could not send the signal: %v", err)
	}

	select {
	case <-chDone:
	case <-time.After(testWaitTimeout):
		t.Fatalf("the done channel was not signaled after the signal")
	}
}

func Test_initProm_disabled(t *testing.T) {
	tests := []struct {
		name string

		endpoint string
		listen   string
	}{
		{
			name:     "both empty",
			endpoint: "",
			listen:   "",
		},
		{
			name:     "listen address is empty",
			endpoint: newEndpoint(),
			listen:   "",
		},
		{
			name:     "endpoint is empty",
			endpoint: "",
			listen:   "127.0.0.1:0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initProm(tt.endpoint, tt.listen)

			if tt.endpoint == "" {
				return
			}

			req, err := http.NewRequest(http.MethodGet, tt.endpoint, nil)
			if err != nil {
				t.Fatalf("could not create the request: %v", err)
			}
			if _, pattern := http.DefaultServeMux.Handler(req); pattern != "" {
				t.Errorf("expected no handler to be registered, got pattern: %q", pattern)
			}
		})
	}
}

func Test_initProm_enabled(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a port: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	endpoint := newEndpoint()
	initProm(endpoint, addr)

	url := "http://" + addr + endpoint
	deadline := time.Now().Add(testWaitTimeout)
	for {
		res, errGet := http.Get(url)
		if errGet == nil {
			body, _ := io.ReadAll(res.Body)
			_ = res.Body.Close()

			if res.StatusCode != http.StatusOK {
				t.Fatalf("expected status 200, got: %d", res.StatusCode)
			}
			if !strings.Contains(string(body), "go_goroutines") {
				t.Errorf("expected the prometheus metrics in the body, got: %.200s", string(body))
			}
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("the metrics server did not answer in time. last error: %v", errGet)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func Test_initProm_listenError(t *testing.T) {
	// keep the port occupied so that the metrics server cannot listen on it
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a port: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })

	ch := hookLog(t, "Prometheus server error")
	initProm(newEndpoint(), l.Addr().String())

	select {
	case msg := <-ch:
		if !strings.Contains(msg, "address already in use") {
			t.Errorf("unexpected log message: %s", msg)
		}
	case <-time.After(testWaitTimeout):
		t.Fatalf("the listen error was not logged")
	}
}

func Test_initCache(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		mr := miniredis.RunT(t)
		setConfig(t, config.Config{RedisAddress: mr.Addr()})

		res, err := initCache()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatalf("expected a cache handler, got nil")
		}
		if mr.TotalConnectionCount() < 1 {
			t.Errorf("expected the cache server to be connected, count: %d", mr.TotalConnectionCount())
		}
	})

	t.Run("server is down", func(t *testing.T) {
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()
		setConfig(t, config.Config{RedisAddress: addr})

		res, err := initCache()
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}
		if res != nil {
			t.Errorf("expected no handler on error, got: %v", res)
		}
		if !strings.HasPrefix(err.Error(), "cache connect error") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func Test_runSubscribe(t *testing.T) {
	queue := string(commonoutline.QueueNameTransferSubscribe)
	exchange := string(commonoutline.QueueNameEvent)
	errSentinel := errors.New("injected failure")

	t.Run("started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		consumed := make(chan struct{})
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
		)
		mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(nil).MinTimes(1)
		mockSock.EXPECT().ConsumeMessage(gomock.Any(), queue, serviceName, false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, numWorkers int, cb sock.CbMsgConsume) error {
				close(consumed)
				return nil
			})

		if err := runSubscribe(serviceName, mockSock, mockTransfer); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		select {
		case <-consumed:
		case <-time.After(testWaitTimeout):
			t.Fatalf("the subscribe consumer was not started")
		}
	})

	t.Run("queue create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		// no other call is expected after the failure
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errSentinel)

		err := runSubscribe(serviceName, mockSock, mockTransfer)
		checkRunError(t, err, "could not run the subscribe handler", errSentinel)
	})

	t.Run("topic create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(errSentinel),
		)

		err := runSubscribe(serviceName, mockSock, mockTransfer)
		checkRunError(t, err, "could not run the subscribe handler", errSentinel)
	})

	t.Run("second bind fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		// the third bind and the consumer must not be reached after the failure
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
			mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(nil),
			mockSock.EXPECT().QueueBind(queue, gomock.Any(), exchange, false, nil).Return(errSentinel),
		)

		err := runSubscribe(serviceName, mockSock, mockTransfer)
		checkRunError(t, err, "could not run the subscribe handler", errSentinel)
	})
}

func Test_runRequestListen(t *testing.T) {
	queue := string(commonoutline.QueueNameTransferRequest)
	errSentinel := errors.New("injected failure")

	t.Run("started", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		consumed := make(chan struct{})
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil),
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, gomock.Any(), false, false, false, 10, gomock.Any()).
				DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
					close(consumed)
					return nil
				}),
		)

		if err := runRequestListen(mockSock, mockTransfer); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		select {
		case <-consumed:
		case <-time.After(testWaitTimeout):
			t.Fatalf("the request consumer was not started")
		}
	})

	t.Run("queue create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockTransfer := transferhandler.NewMockTransferHandler(mc)

		// the consumer must not be started after the failure
		mockSock.EXPECT().QueueCreate(queue, "normal").Return(errSentinel)

		err := runRequestListen(mockSock, mockTransfer)
		checkRunError(t, err, "could not run the listen handler", errSentinel)
	})
}

func checkRunError(t *testing.T, err error, prefix string, cause error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Errorf("expected the error to start with %q, got: %v", prefix, err)
	}
	if !errors.Is(err, cause) {
		t.Errorf("expected the cause to be kept in the chain, got: %v", err)
	}
}

func Test_runDaemon_databaseFailure(t *testing.T) {
	// the prometheus server must stay off
	setConfig(t, config.Config{
		DatabaseDSN:             testUnreachableDSN,
		PrometheusEndpoint:      "",
		PrometheusListenAddress: "",
	})
	t.Cleanup(func() { signal.Stop(chSigs) })

	err := runDaemon()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	msg := err.Error()
	for _, part := range []string{
		"could not connect to the database",
		"database ping error",
	} {
		if !strings.Contains(msg, part) {
			t.Errorf("expected %q in the error, got: %s", part, msg)
		}
	}
	if !errors.Is(err, syscall.ENOENT) {
		t.Errorf("expected the cause to be a missing socket, got: %v", err)
	}
}
