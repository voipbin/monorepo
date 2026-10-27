package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conversation-manager/internal/config"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

var (
	errConnectTest = errors.New("connect failure")
	errMigrateTest = errors.New("migrate failure")
)

const (
	requestQueue   = "bin-manager.conversation-manager.request"
	subscribeQueue = "bin-manager.conversation-manager.subscribe"
)

// seamGlobals isolates a test from the process wide state that the start path of the service
// changes: the global configuration, the default metrics registry and http mux, the signal
// subscription and the done channel. Tests that use it must not run in parallel.
func seamGlobals(t *testing.T) {
	t.Helper()

	// the global configuration is loaded once per process; a later load does not overwrite the
	// fields that this test sets
	config.LoadGlobalConfig()
	prevConfig := *config.Get()
	config.Get().PrometheusEndpoint = "/metrics"
	config.Get().PrometheusListenAddress = "127.0.0.1:0"
	config.Get().DatabaseDSN = "seam"

	prevMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()

	drain := func() {
		for {
			select {
			case <-chDone:
			case <-chSigs:
			default:
				return
			}
		}
	}
	drain()

	t.Cleanup(func() {
		signal.Stop(chSigs)
		drain()
		http.DefaultServeMux = prevMux
		*config.Get() = prevConfig
	})
}

// seamRegistry gives the next handler initialization a fresh metrics registry, because the request
// handler registers the same collectors on every initialization.
func seamRegistry(t *testing.T) {
	t.Helper()

	reg := prometheus.NewRegistry()
	prevRegisterer, prevGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
	t.Cleanup(func() { prometheus.DefaultRegisterer, prometheus.DefaultGatherer = prevRegisterer, prevGatherer })
}

func newSeamDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open the database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// seamEngine is a fake migration engine that counts the calls.
type seamEngine struct {
	upErr error

	up     atomic.Int32
	closed atomic.Int32
}

func (e *seamEngine) Version() (uint, bool, error) { return 1, false, nil }
func (e *seamEngine) Up() error                    { e.up.Add(1); return e.upErr }
func (e *seamEngine) Steps(int) error              { return nil }
func (e *seamEngine) Force(int) error              { return nil }
func (e *seamEngine) Close() (error, error)        { e.closed.Add(1); return nil, nil }

// seamSock is a strict broker mock for the start path. The fields select the failing step.
type seamSock struct {
	listenQueueErr    error
	subscribeQueueErr error

	rpcStarted chan struct{}
	msgStarted chan struct{}
}

// newSeamSock registers the expectations of run() up to the step that fails. The consumers are
// started in goroutines, so their start is reported through channels.
func newSeamSock(mc *gomock.Controller, cfg seamSock) (*sockhandler.MockSockHandler, *seamSock) {
	cfg.rpcStarted = make(chan struct{}, 1)
	cfg.msgStarted = make(chan struct{}, 1)

	m := sockhandler.NewMockSockHandler(mc)
	m.EXPECT().Connect().Times(1)

	// the notify handler declares the global event exchange when it is created
	m.EXPECT().TopicCreateWithKind("bin-manager.event", "topic").Return(nil).Times(1)

	m.EXPECT().QueueCreate(requestQueue, "normal").Return(cfg.listenQueueErr).Times(1)
	if cfg.listenQueueErr != nil {
		return m, &cfg
	}

	m.EXPECT().ConsumeRPC(gomock.Any(), requestQueue, "conversation-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			cfg.rpcStarted <- struct{}{}
			return nil
		}).Times(1)

	m.EXPECT().QueueCreate(subscribeQueue, "normal").Return(cfg.subscribeQueueErr).Times(1)
	if cfg.subscribeQueueErr != nil {
		return m, &cfg
	}

	m.EXPECT().TopicCreateWithKind("bin-manager.event", "topic").Return(nil).Times(1)
	m.EXPECT().QueueBind(subscribeQueue, gomock.Any(), "bin-manager.event", false, nil).Return(nil).Times(4)
	m.EXPECT().ConsumeMessage(gomock.Any(), subscribeQueue, "conversation-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			cfg.msgStarted <- struct{}{}
			return nil
		}).Times(1)

	return m, &cfg
}

func waitStarted(t *testing.T, ch chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("the %s consumer was not started", what)
	}
}

func Test_run(t *testing.T) {
	tests := []struct {
		name string

		sock seamSock

		expectErr     error
		expectRPC     bool
		expectMessage bool
	}{
		{
			name:          "starts the listener and the subscriber",
			expectRPC:     true,
			expectMessage: true,
		},
		{
			name:      "a listen queue failure stops before the subscriber",
			sock:      seamSock{listenQueueErr: errBrokerTest},
			expectErr: errBrokerTest,
		},
		{
			name:      "a subscribe queue failure keeps the listener running",
			sock:      seamSock{subscribeQueueErr: errBrokerTest},
			expectErr: errBrokerTest,
			expectRPC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			seamGlobals(t)
			seamRegistry(t)

			mockSock, cfg := newSeamSock(mc, tt.sock)
			t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock }))

			mockCache := cachehandler.NewMockCacheHandler(mc)

			err := run(newSeamDB(t), mockCache, config.Get())
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectRPC {
				waitStarted(t, cfg.rpcStarted, "rpc")
			}
			if tt.expectMessage {
				waitStarted(t, cfg.msgStarted, "event")
			}
		})
	}
}

func Test_initProm(t *testing.T) {
	seamGlobals(t)

	initProm("/seam-metrics", "127.0.0.1:0")

	req, err := http.NewRequest(http.MethodGet, "/seam-metrics", nil)
	if err != nil {
		t.Fatalf("could not create the request: %v", err)
	}
	if _, pattern := http.DefaultServeMux.Handler(req); pattern != "/seam-metrics" {
		t.Errorf("Wrong match. expect: the metrics path is registered, got pattern: %q", pattern)
	}

	other, err := http.NewRequest(http.MethodGet, "/other", nil)
	if err != nil {
		t.Fatalf("could not create the request: %v", err)
	}
	if _, pattern := http.DefaultServeMux.Handler(other); pattern != "" {
		t.Errorf("Wrong match. expect: no other path is registered, got pattern: %q", pattern)
	}
}

func Test_runMain_startPath(t *testing.T) {
	type env struct {
		engine        *seamEngine
		engineOpened  atomic.Int32
		sockFactory   atomic.Int32
		connectCalled atomic.Int32
	}

	// setup installs the injected dependencies. A cache that is not running makes the cache
	// connection fail.
	setup := func(t *testing.T, mc *gomock.Controller, connectErr error, upErr error, cacheRunning bool, mockSock *sockhandler.MockSockHandler) *env {
		e := &env{engine: &seamEngine{upErr: upErr}}

		mr := miniredis.RunT(t)
		config.Get().RedisAddress = mr.Addr()
		if !cacheRunning {
			mr.Close()
		}

		t.Cleanup(databasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
			e.connectCalled.Add(1)
			if connectErr != nil {
				return nil, connectErr
			}
			return newSeamDB(t), nil
		}))
		t.Cleanup(migrationhandler.SetEngineFunc(func(string, fs.FS, string) (migrationhandler.Engine, uint, error) {
			e.engineOpened.Add(1)
			return e.engine, 1, nil
		}))
		t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler {
			e.sockFactory.Add(1)
			return mockSock
		}))

		return e
	}

	t.Run("a database connection failure stops before the migrations", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		seamGlobals(t)
		seamRegistry(t)
		e := setup(t, mc, errConnectTest, nil, true, nil)

		err := runMain(&cobra.Command{}, nil)
		if !errors.Is(err, errConnectTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errConnectTest, err)
		}
		if e.engineOpened.Load() != 0 || e.sockFactory.Load() != 0 {
			t.Errorf("Wrong match. expect: no engine and no broker, got: engine %d broker %d", e.engineOpened.Load(), e.sockFactory.Load())
		}
	})

	t.Run("a migration failure stops before the cache connection", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		seamGlobals(t)
		seamRegistry(t)

		// the cache is not running, so a cache connection attempt would return a different error
		e := setup(t, mc, nil, errMigrateTest, false, nil)

		err := runMain(&cobra.Command{}, nil)
		if !errors.Is(err, errMigrateTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errMigrateTest, err)
		}
		if e.engine.up.Load() != 1 || e.engine.closed.Load() != 1 {
			t.Errorf("Wrong match. expect: one up and one close, got: up %d close %d", e.engine.up.Load(), e.engine.closed.Load())
		}
		if e.sockFactory.Load() != 0 {
			t.Errorf("Wrong match. expect: no broker, got: %d", e.sockFactory.Load())
		}
	})

	t.Run("a cache connection failure stops after the migrations and before the broker", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		seamGlobals(t)
		seamRegistry(t)
		e := setup(t, mc, nil, nil, false, nil)

		err := runMain(&cobra.Command{}, nil)
		if err == nil || errors.Is(err, errMigrateTest) || errors.Is(err, errConnectTest) {
			t.Errorf("Wrong match. expect: the cache error, got: %v", err)
		}
		if e.engine.up.Load() != 1 || e.engine.closed.Load() != 1 {
			t.Errorf("Wrong match. expect: one up and one close, got: up %d close %d", e.engine.up.Load(), e.engine.closed.Load())
		}
		if e.sockFactory.Load() != 0 {
			t.Errorf("Wrong match. expect: no broker, got: %d", e.sockFactory.Load())
		}
	})

	t.Run("runs until a termination signal arrives", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		seamGlobals(t)
		seamRegistry(t)

		mockSock, cfg := newSeamSock(mc, seamSock{})
		e := setup(t, mc, nil, nil, true, mockSock)

		done := make(chan error, 1)
		go func() { done <- runMain(&cobra.Command{}, nil) }()

		// the service is fully started once both consumers run; it must then keep running
		waitStarted(t, cfg.rpcStarted, "rpc")
		waitStarted(t, cfg.msgStarted, "event")
		select {
		case err := <-done:
			t.Fatalf("Wrong match. expect: still running, got: returned with %v", err)
		case <-time.After(300 * time.Millisecond):
		}

		chSigs <- os.Interrupt

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("the service did not stop after the signal")
		}

		if e.engine.up.Load() != 1 || e.engine.closed.Load() != 1 || e.sockFactory.Load() != 1 {
			t.Errorf("Wrong match. expect: one up, one close, one broker, got: up %d close %d broker %d", e.engine.up.Load(), e.engine.closed.Load(), e.sockFactory.Load())
		}
	})
}
