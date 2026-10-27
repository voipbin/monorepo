package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-outdial-manager/internal/config"
	"monorepo/bin-outdial-manager/migrations"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

const waitTimeout = 5 * time.Second

var promPathCounter atomic.Int32

// uniquePromPath returns a metrics path that no other test registers on the default mux, because
// the same pattern registered twice panics.
func uniquePromPath() string {
	return fmt.Sprintf("/test-metrics-%d", promPathCounter.Add(1))
}

// fakeEngine is a migration engine that counts the calls and fails Up with the configured error.
type fakeEngine struct {
	upErr error

	// what the factory was asked to open
	dsn          string
	versionTable string

	upCalls    atomic.Int32
	closeCalls atomic.Int32
}

func (f *fakeEngine) Version() (uint, bool, error) { return 0, false, nil }
func (f *fakeEngine) Up() error                    { f.upCalls.Add(1); return f.upErr }
func (f *fakeEngine) Steps(n int) error            { return nil }
func (f *fakeEngine) Force(version int) error      { return nil }
func (f *fakeEngine) Close() (error, error)        { f.closeCalls.Add(1); return nil, nil }

type managerEnv struct {
	mc     *gomock.Controller
	sock   *sockhandler.MockSockHandler
	engine *fakeEngine
	redis  *miniredis.Miniredis

	rabbitAddress atomic.Value // the address the sock factory was called with
	consumed      chan string
}

// setupManager points the process wide state at test doubles until the test ends: the config (in
// the global config and in viper, because LoadGlobalConfig reads viper once per process), the
// database connection, the migration engine, the sock factory and the metric registry.
func setupManager(t *testing.T) *managerEnv {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	mr := miniredis.RunT(t)
	promPath := uniquePromPath()

	cfg := config.Get()
	orig := *cfg
	t.Cleanup(func() { *cfg = orig })
	cfg.DatabaseDSN = "test-user@unix(/nonexistent-test-dir/none.sock)/test_db" // never dialed
	cfg.PrometheusEndpoint = promPath
	cfg.PrometheusListenAddress = "127.0.0.1:0"
	cfg.RabbitMQAddress = "test-rabbit"
	cfg.RedisAddress = mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	viper.Set("database_dsn", cfg.DatabaseDSN)
	viper.Set("prometheus_endpoint", cfg.PrometheusEndpoint)
	viper.Set("prometheus_listen_address", cfg.PrometheusListenAddress)
	viper.Set("rabbitmq_address", cfg.RabbitMQAddress)
	viper.Set("redis_address", cfg.RedisAddress)
	viper.Set("redis_password", "")
	viper.Set("redis_database", 0)

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create the database mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	t.Cleanup(commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) { return db, nil }))

	engine := &fakeEngine{}
	t.Cleanup(migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (migrationhandler.Engine, uint, error) {
		engine.dsn, engine.versionTable = dsn, versionTable
		return engine, 1, nil
	}))

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	origRegisterer := prometheus.DefaultRegisterer
	t.Cleanup(func() { prometheus.DefaultRegisterer = origRegisterer })
	// the request handler registers its metrics without a guard, so every handler gets a new registry
	env := &managerEnv{mc: mc, sock: mockSock, engine: engine, redis: mr, consumed: make(chan string, 1)}
	t.Cleanup(sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		env.rabbitAddress.Store(serverURI)
		prometheus.DefaultRegisterer = prometheus.NewRegistry()
		return mockSock
	}))

	return env
}

// expectRun sets the calls that run makes on the sock handler: connect, the event exchange, the
// listen queue and the consumer. The consumer starts in a goroutine, so waitConsumer must be
// called before the controller finishes.
func (e *managerEnv) expectRun() {
	e.sock.EXPECT().Connect()
	e.sock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil)
	e.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameOutdialRequest), "normal").Return(nil)
	e.sock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, queue, _ string, _, _, _ bool, _ int, _ any) error {
			e.consumed <- queue
			return nil
		})
}

func (e *managerEnv) waitConsumer(t *testing.T) {
	t.Helper()

	select {
	case got := <-e.consumed:
		if got != string(commonoutline.QueueNameOutdialRequest) {
			t.Errorf("expected the consumer on %q, got %q", commonoutline.QueueNameOutdialRequest, got)
		}
		if addr := e.rabbitAddress.Load(); addr != "test-rabbit" {
			t.Errorf("expected the sock handler on the configured address, got: %v", addr)
		}
	case <-time.After(waitTimeout):
		t.Fatalf("the consumer was not started")
	}
}

// releaseSignalGoroutine ends the goroutine that initSignal started and leaves the channels empty,
// so a later test never loses the value it sends to a goroutine of this test.
func releaseSignalGoroutine(t *testing.T) {
	t.Helper()

	chSigs <- syscall.SIGTERM
	select {
	case <-chDone:
	case <-time.After(waitTimeout):
		t.Fatalf("the signal goroutine did not finish")
	}
	signal.Stop(chSigs)
}

func closedAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func Test_createDBHandler(t *testing.T) {
	t.Run("migration failure stops before the cache", func(t *testing.T) {
		env := setupManager(t)
		env.engine.upErr = errors.New("test migration failure")
		config.Get().RedisAddress = closedAddress(t) // reached only when the order is wrong

		res, err := createDBHandler()
		if res != nil {
			t.Errorf("expected no handler, got: %v", res)
		}
		if err == nil || !strings.Contains(err.Error(), "test migration failure") {
			t.Errorf("expected the migration error, got: %v", err)
		}
		if got := env.engine.upCalls.Load(); got != 1 {
			t.Errorf("expected one migration run, got: %d", got)
		}
	})

	t.Run("cache failure after the migrations", func(t *testing.T) {
		env := setupManager(t)
		config.Get().RedisAddress = closedAddress(t)

		res, err := createDBHandler()
		if res != nil {
			t.Errorf("expected no handler, got: %v", res)
		}
		if err == nil {
			t.Errorf("expected the cache connection error, got nil")
		}
		if got := env.engine.upCalls.Load(); got != 1 {
			t.Errorf("expected the migrations to run before the cache, got: %d runs", got)
		}
	})

	t.Run("success", func(t *testing.T) {
		env := setupManager(t)

		res, err := createDBHandler()
		if err != nil || res == nil {
			t.Fatalf("expected a handler, got: %v, %v", res, err)
		}
		if env.engine.upCalls.Load() != 1 || env.engine.closeCalls.Load() != 1 {
			t.Errorf("expected one migration run and one close, got: %d, %d", env.engine.upCalls.Load(), env.engine.closeCalls.Load())
		}
		if env.engine.dsn != config.Get().DatabaseDSN || env.engine.versionTable != migrations.VersionTable {
			t.Errorf("expected the migrations on the configured database and the service version table, got: %q, %q", env.engine.dsn, env.engine.versionTable)
		}
	})
}

func Test_run(t *testing.T) {
	env := setupManager(t)
	env.expectRun()

	if err := run(dbhandler.NewMockDBHandler(env.mc)); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}
	env.waitConsumer(t)
}

func Test_runDaemon(t *testing.T) {
	t.Run("database failure is returned", func(t *testing.T) {
		env := setupManager(t)
		env.engine.upErr = errors.New("test migration failure")
		t.Cleanup(func() { releaseSignalGoroutine(t) })

		err := runDaemon()
		if err == nil || !strings.Contains(err.Error(), "could not connect to the database or failed to initiate the cachehandler") ||
			!strings.Contains(err.Error(), "test migration failure") {
			t.Errorf("expected the wrapped startup error, got: %v", err)
		}
		if len(chDone) != 0 {
			t.Errorf("expected no stop signal, got one")
		}
	})

	t.Run("success runs until the stop signal", func(t *testing.T) {
		env := setupManager(t)
		env.expectRun()
		t.Cleanup(func() { signal.Stop(chSigs) })

		returned := make(chan error, 1)
		go func() { returned <- runDaemon() }()
		env.waitConsumer(t) // the service is up

		select {
		case err := <-returned:
			t.Fatalf("the daemon returned before the stop signal: %v", err)
		case <-time.After(100 * time.Millisecond):
		}

		chSigs <- syscall.SIGTERM
		select {
		case err := <-returned:
			if err != nil {
				t.Fatalf("expected nil, got: %v", err)
			}
		case <-time.After(waitTimeout):
			t.Fatalf("the daemon did not stop after the signal")
		}

		if env.engine.upCalls.Load() != 1 {
			t.Errorf("expected the migrations to run once, got: %d", env.engine.upCalls.Load())
		}

		// the metrics endpoint of the configuration is registered
		req := httptest.NewRequest(http.MethodGet, config.Get().PrometheusEndpoint, nil)
		if _, pattern := http.DefaultServeMux.Handler(req); pattern != config.Get().PrometheusEndpoint {
			t.Errorf("expected the handler on %q, got pattern %q", config.Get().PrometheusEndpoint, pattern)
		}
	})
}

func Test_initSignal(t *testing.T) {
	initSignal()
	t.Cleanup(func() { signal.Stop(chSigs) })

	chSigs <- syscall.SIGHUP
	select {
	case done := <-chDone:
		if !done {
			t.Errorf("expected true on the done channel")
		}
	case <-time.After(waitTimeout):
		t.Fatalf("the signal was not turned into a stop")
	}
}

func Test_initProm(t *testing.T) {
	path := uniquePromPath()
	initProm(path, "127.0.0.1:0")

	req := httptest.NewRequest(http.MethodGet, path, nil)
	handler, pattern := http.DefaultServeMux.Handler(req)
	if pattern != path {
		t.Fatalf("expected the handler on %q, got pattern %q", path, pattern)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "go_goroutines") {
		t.Errorf("expected the metrics exposition, got: %d %q", rec.Code, rec.Body.String())
	}
}

// errorHook forwards the messages that contain the marker.
type errorHook struct {
	marker string
	found  chan string
	once   sync.Once
}

func (h *errorHook) Levels() []logrus.Level { return []logrus.Level{logrus.ErrorLevel} }
func (h *errorHook) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, h.marker) {
		h.once.Do(func() { h.found <- e.Message })
	}
	return nil
}

func Test_initProm_listenFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	t.Cleanup(func() { _ = occupied.Close() })

	hook := &errorHook{marker: "Prometheus server error", found: make(chan string, 1)}
	origHooks := logrus.StandardLogger().ReplaceHooks(logrus.LevelHooks{})
	logrus.StandardLogger().AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(origHooks) })

	initProm(uniquePromPath(), occupied.Addr().String())

	select {
	case msg := <-hook.found:
		if !strings.Contains(msg, "address already in use") {
			t.Errorf("expected the address error, got: %q", msg)
		}
	case <-time.After(waitTimeout):
		t.Fatalf("the listen failure was not logged")
	}
}
