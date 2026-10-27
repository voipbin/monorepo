package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-conference-manager/internal/config"
	"monorepo/bin-conference-manager/pkg/cachehandler"
)

// testRedis is the stand-in cache server. The config is loaded once, so its address is set before the first load.
var testRedis *miniredis.Miniredis

func TestMain(m *testing.M) {
	testRedis = miniredis.NewMiniRedis()
	if err := testRedis.Start(); err != nil {
		panic(err)
	}

	// viper.Set wins over flags and environment variables of the host.
	viper.Set("database_dsn", "not-a-valid-dsn")
	viper.Set("rabbitmq_address", "amqp://localhost")
	viper.Set("prometheus_endpoint", "/metrics")
	viper.Set("prometheus_listen_address", "127.0.0.1:0")
	viper.Set("redis_address", testRedis.Addr())
	viper.Set("redis_password", "")
	viper.Set("redis_database", 1)
	config.LoadGlobalConfig()

	code := m.Run()
	testRedis.Close()
	os.Exit(code)
}

// fakeEngine records the calls of the migration runner.
type fakeEngine struct {
	upErr error

	upCalls    int
	closeCalls int
}

func (f *fakeEngine) Version() (uint, bool, error) { return 1, false, nil }
func (f *fakeEngine) Up() error                    { f.upCalls++; return f.upErr }
func (f *fakeEngine) Steps(int) error              { return nil }
func (f *fakeEngine) Force(int) error              { return nil }
func (f *fakeEngine) Close() (error, error)        { f.closeCalls++; return nil, nil }

// failures selects the calls of the message broker that fail. The number is the 1 based index
// of the call, 0 means that the call never fails.
type failures struct {
	topic       int
	queueCreate int
	queueBind   int
}

// env is the wired environment of a start path test.
type env struct {
	t *testing.T

	db      *sql.DB
	engine  *fakeEngine
	consume chan string // names of the consume calls made from the goroutines of the handlers

	topicCalls, queueCalls, bindCalls atomic.Int64
}

// newEnv installs the injection points of the shared library and the isolation of the global state.
// All of them are process wide, so the tests that use it never run in parallel.
func newEnv(t *testing.T, f failures) *env {
	t.Helper()

	e := &env{t: t, engine: &fakeEngine{}, consume: make(chan string, 8)}

	// every construction of the request handler registers metrics, and the prometheus endpoint is
	// registered on the default mux, so each test gets its own registry and mux.
	prevReg, prevGather := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
	prevMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = prevReg, prevGather
		http.DefaultServeMux = prevMux
	})

	testRedis.FlushAll()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("could not open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	e.db = db

	restoreDB := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return db, nil })
	t.Cleanup(restoreDB)

	restoreEngine := migrationhandler.SetEngineFunc(func(string, fs.FS, string) (migrationhandler.Engine, uint, error) {
		return e.engine, 1, nil
	})
	t.Cleanup(restoreEngine)

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().AnyTimes()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
		return failAt(&e.topicCalls, f.topic, "topic")
	}).AnyTimes()
	mockSock.EXPECT().QueueCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(string, string) error {
		return failAt(&e.queueCalls, f.queueCreate, "queue create")
	}).AnyTimes()
	mockSock.EXPECT().QueueBind(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_, _, _ string, _ bool, _ any) error {
		return failAt(&e.bindCalls, f.queueBind, "queue bind")
	}).AnyTimes()
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
		e.consume <- "rpc"
		return nil
	}).AnyTimes()
	mockSock.EXPECT().ConsumeMessage(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgConsume) error {
		e.consume <- "message"
		return nil
	}).AnyTimes()

	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)

	return e
}

// failAt counts the call and returns an error when it is the call that must fail.
func failAt(counter *atomic.Int64, failIndex int, name string) error {
	if int(counter.Add(1)) == failIndex {
		return errors.New("synthetic " + name + " failure")
	}
	return nil
}

// waitConsume waits until both consumers have been started by their goroutines.
func (e *env) waitConsume() {
	e.t.Helper()

	got := map[string]bool{}
	timeout := time.After(5 * time.Second)
	for len(got) < 2 {
		select {
		case name := <-e.consume:
			got[name] = true
		case <-timeout:
			e.t.Fatalf("the consumers were not started, got: %v", got)
		}
	}
}

func newCache(t *testing.T) cachehandler.CacheHandler {
	t.Helper()

	cfg := config.Get()
	cache := cachehandler.NewHandler(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDatabase)
	if err := cache.Connect(); err != nil {
		t.Fatalf("could not connect to the stand-in cache server: %v", err)
	}
	return cache
}

// runMainAsync starts runMain and returns the channel of its result.
func runMainAsync() <-chan error {
	res := make(chan error, 1)
	go func() { res <- runMain(rootCmd, nil) }()
	return res
}

// runMainFailing runs a start that is expected to fail and returns its error. A start that does not
// fail would wait for a signal, so the wait is bounded.
func runMainFailing(t *testing.T) error {
	t.Helper()

	select {
	case err := <-runMainAsync():
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("runMain did not return an error")
		return nil
	}
}

// releaseSignalHandler ends the signal handler goroutine that runMain left behind and clears the done channel.
// Every call of initSignal starts one goroutine that waits for one value on chSigs.
func releaseSignalHandler(t *testing.T) {
	t.Helper()

	signal.Stop(chSigs)
	chSigs <- syscall.SIGTERM
	select {
	case <-chDone:
	case <-time.After(5 * time.Second):
		t.Fatalf("the signal handler did not finish")
	}
}

func Test_runMain(t *testing.T) {
	t.Run("starts, applies the migrations and stops on a signal", func(t *testing.T) {
		e := newEnv(t, failures{})

		res := runMainAsync()
		e.waitConsume()

		select {
		case err := <-res:
			t.Fatalf("Wrong match. expect: still running, got: %v", err)
		case <-time.After(50 * time.Millisecond):
		}

		// a value on chSigs stands for a real signal
		signal.Stop(chSigs)
		chSigs <- syscall.SIGTERM

		select {
		case err := <-res:
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("runMain did not stop")
		}

		if e.engine.upCalls != 1 || e.engine.closeCalls != 1 {
			t.Errorf("Wrong match. expect: one up and one close of the migration engine, got: %d, %d", e.engine.upCalls, e.engine.closeCalls)
		}
	})

	t.Run("the migration failure stops the start", func(t *testing.T) {
		e := newEnv(t, failures{})
		e.engine.upErr = errors.New("synthetic migration failure")
		t.Cleanup(func() { releaseSignalHandler(t) })

		err := runMainFailing(t)
		if err == nil || !strings.Contains(err.Error(), "synthetic migration failure") {
			t.Fatalf("Wrong match. expect: migration error, got: %v", err)
		}
		if len(e.consume) != 0 {
			t.Errorf("Wrong match. expect: no consumer started, got: %d", len(e.consume))
		}
	})

	t.Run("the database failure stops the start before the migrations", func(t *testing.T) {
		e := newEnv(t, failures{})
		restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return nil, errors.New("synthetic connect failure") })
		t.Cleanup(restore)
		t.Cleanup(func() { releaseSignalHandler(t) })

		err := runMainFailing(t)
		if err == nil || !strings.Contains(err.Error(), "synthetic connect failure") {
			t.Fatalf("Wrong match. expect: connect error, got: %v", err)
		}
		if e.engine.upCalls != 0 {
			t.Errorf("Wrong match. expect: no migration, got: %d", e.engine.upCalls)
		}
	})

	t.Run("an unreachable cache stops the start", func(t *testing.T) {
		e := newEnv(t, failures{})
		t.Cleanup(func() { releaseSignalHandler(t) })

		testRedis.Close()
		t.Cleanup(func() {
			if err := testRedis.Restart(); err != nil {
				t.Fatalf("could not restart the stand-in cache server: %v", err)
			}
		})

		err := runMainFailing(t)
		if err == nil {
			t.Fatalf("Wrong match. expect: cache error, got: nil")
		}
		if len(e.consume) != 0 {
			t.Errorf("Wrong match. expect: no consumer started, got: %d", len(e.consume))
		}
	})
}

func Test_run(t *testing.T) {
	tests := []struct {
		name string

		failures failures

		expectErr      string
		expectConsumes bool
	}{
		{
			name:           "starts both consumers",
			expectConsumes: true,
		},
		{
			name:      "the listen queue failure is returned",
			failures:  failures{queueCreate: 1},
			expectErr: "could not declare the queue for listenHandler",
		},
		{
			name:      "the subscribe queue failure is returned",
			failures:  failures{queueCreate: 2},
			expectErr: "could not declare the queue for listenHandler",
		},
		{
			name:      "the global topic failure of the subscribe start is returned",
			failures:  failures{topic: 2},
			expectErr: "could not declare the global topic exchange",
		},
		{
			name:      "a failing topic binding is returned",
			failures:  failures{queueBind: 1},
			expectErr: "could not bind the topic pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, tt.failures)
			cache := newCache(t)

			err := run(e.db, cache, config.Get())
			if tt.expectErr == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: %s, got: %v", tt.expectErr, err)
			}

			if tt.expectConsumes {
				e.waitConsume()
			}
		})
	}
}

func Test_runListen_runSubscribe(t *testing.T) {
	t.Run("listen start failure", func(t *testing.T) {
		newEnv(t, failures{queueCreate: 1})
		sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, "")

		if err := runListen(sockHandler, nil, nil); err == nil || !strings.Contains(err.Error(), "could not declare the queue") {
			t.Fatalf("Wrong match. expect: queue error, got: %v", err)
		}
	})

	t.Run("subscribe start failure", func(t *testing.T) {
		newEnv(t, failures{queueCreate: 1})
		sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, "")

		if err := runSubscribe(sockHandler, nil, nil); err == nil || !strings.Contains(err.Error(), "could not declare the queue") {
			t.Fatalf("Wrong match. expect: queue error, got: %v", err)
		}
	})
}

func Test_initProm(t *testing.T) {
	newEnv(t, failures{})

	initProm("/probe", "127.0.0.1:0")

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rec := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, rec.Code)
	}
}
