package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-message-manager/internal/config"
	"monorepo/bin-message-manager/migrations"
)

const (
	testRabbitAddress = "test-rabbit-address"
	testQueueType     = "normal"
	testConsumerName  = "message-manager"
	testEngineMax     = uint(7)
)

var testDBSeq atomic.Int64

// fakeEngine records the calls of the migration runner and returns the configured errors.
type fakeEngine struct {
	version    uint
	versionErr error
	upErr      error

	upCalls    atomic.Int32
	closeCalls atomic.Int32
}

func (f *fakeEngine) Version() (uint, bool, error) { return f.version, false, f.versionErr }
func (f *fakeEngine) Up() error {
	f.upCalls.Add(1)
	return f.upErr
}
func (f *fakeEngine) Steps(int) error { return nil }
func (f *fakeEngine) Force(int) error { return nil }
func (f *fakeEngine) Close() (error, error) {
	f.closeCalls.Add(1)
	return nil, nil
}

// serviceEnv holds the test doubles that stand in for the database, the migration engine, the broker and the cache.
type serviceEnv struct {
	db     *sql.DB
	engine *fakeEngine
	sock   *sockhandler.MockSockHandler
	redis  *miniredis.Miniredis

	connectDSN    atomic.Value // string
	engineDSN     atomic.Value // string
	engineTable   atomic.Value // string
	engineFSIsNil atomic.Bool
	rabbitURI     atomic.Value // string

	connectErr error
	engineErr  error

	consumed chan struct{}
}

// replaceRegisterer gives the next handler construction its own metrics registry. The request handler and the
// circuit breaker register collectors without a guard, so a second construction in one process would panic.
func replaceRegisterer(t *testing.T) {
	t.Helper()

	orig := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = orig })
}

// newServiceEnv installs the test doubles through the setters of the common handler and a config that opens no port.
// Tests that use it must not run in parallel, because the setters, the config and the signal channels are process globals.
func newServiceEnv(t *testing.T) *serviceEnv {
	t.Helper()

	prepareSignalTest(t)
	replaceRegisterer(t)

	origCfg := *config.Get()
	t.Cleanup(func() { *config.Get() = origCfg })

	mr := miniredis.RunT(t)

	// every test gets its own in-memory database name
	name := fmt.Sprintf("file:message_manager_run_%d?mode=memory&cache=shared", testDBSeq.Add(1))
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("Could not open the database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	mc := gomock.NewController(t)
	env := &serviceEnv{
		db:       db,
		engine:   &fakeEngine{},
		sock:     sockhandler.NewMockSockHandler(mc),
		redis:    mr,
		consumed: make(chan struct{}, 1),
	}

	cfg := config.Get()
	cfg.DatabaseDSN = "test-database-dsn"
	cfg.PrometheusEndpoint = ""
	cfg.PrometheusListenAddress = ""
	cfg.RabbitMQAddress = testRabbitAddress
	cfg.RedisAddress = mr.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	cfg.AuthtokenMessagebird = "test-token-a"
	cfg.AuthtokenTelnyx = "test-token-b"

	restoreConnect := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		env.connectDSN.Store(dsn)
		if env.connectErr != nil {
			return nil, env.connectErr
		}
		return env.db, nil
	})
	t.Cleanup(restoreConnect)

	restoreEngine := migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (migrationhandler.Engine, uint, error) {
		env.engineDSN.Store(dsn)
		env.engineTable.Store(versionTable)
		env.engineFSIsNil.Store(fsys == nil)
		if env.engineErr != nil {
			return nil, 0, env.engineErr
		}
		return env.engine, testEngineMax, nil
	})
	t.Cleanup(restoreEngine)

	restoreSock := sockhandler.SetRabbitFactory(func(uri string) sockhandler.SockHandler {
		env.rabbitURI.Store(uri)
		return env.sock
	})
	t.Cleanup(restoreSock)

	return env
}

// expectListen sets the broker calls of a complete start: the connect, the event exchange, the queue and the consumer.
// The consumer runs in a goroutine, so its call is reported through env.consumed.
func (e *serviceEnv) expectListen(queueErr error) {
	e.sock.EXPECT().Connect().Times(1)
	e.sock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil).Times(1)
	e.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameMessageRequest), testQueueType).Return(queueErr).Times(1)

	if queueErr != nil {
		return
	}

	e.sock.EXPECT().ConsumeRPC(gomock.Any(), string(commonoutline.QueueNameMessageRequest), testConsumerName, false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			e.consumed <- struct{}{}
			return nil
		}).Times(1)
}

// waitConsumed waits for the consumer goroutine of the listen handler.
func (e *serviceEnv) waitConsumed(t *testing.T) {
	t.Helper()

	select {
	case <-e.consumed:
	case <-time.After(signalWait):
		t.Fatalf("Wrong match. expect: consumer started, got: timeout")
	}
}

// runServiceWithSignal runs runService with the stop signal already queued. When runService fails before it waits for
// the done channel, the signal handler goroutine started by initSignal still has to be ended, so the helper waits for it.
func runServiceWithSignal(t *testing.T) error {
	t.Helper()

	chSigs <- syscall.SIGTERM
	err := runService()
	if err != nil {
		waitDone(t)
	}
	return err
}

func Test_runService_success(t *testing.T) {
	env := newServiceEnv(t)
	env.expectListen(nil)

	if err := runServiceWithSignal(t); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	env.waitConsumed(t)

	if got := env.connectDSN.Load(); got != "test-database-dsn" {
		t.Errorf("Wrong database dsn. expect: test-database-dsn, got: %v", got)
	}
	if got := env.engineDSN.Load(); got != "test-database-dsn" {
		t.Errorf("Wrong migration dsn. expect: test-database-dsn, got: %v", got)
	}
	if got := env.engineTable.Load(); got != migrations.VersionTable {
		t.Errorf("Wrong version table. expect: %s, got: %v", migrations.VersionTable, got)
	}
	if env.engineFSIsNil.Load() {
		t.Errorf("Wrong migration source. expect: the embedded migrations, got: nil")
	}
	if got := env.rabbitURI.Load(); got != testRabbitAddress {
		t.Errorf("Wrong broker address. expect: %s, got: %v", testRabbitAddress, got)
	}
	if got := env.engine.upCalls.Load(); got != 1 {
		t.Errorf("Wrong migration up count. expect: 1, got: %d", got)
	}
	if got := env.engine.closeCalls.Load(); got != 1 {
		t.Errorf("Wrong migration close count. expect: 1, got: %d", got)
	}

	// runService closes the database it opened when it returns.
	if errPing := env.db.Ping(); errPing == nil {
		t.Errorf("Wrong match. expect: the database is closed, got: open")
	}
}

// Test_runService_waitsForStopSignal checks that runService keeps running after it started the listener and returns
// only after the stop signal. The "still running" check waits a short time, so it can only miss a premature return,
// never fail on a correct one.
func Test_runService_waitsForStopSignal(t *testing.T) {
	env := newServiceEnv(t)
	env.expectListen(nil)

	result := make(chan error, 1)
	go func() { result <- runService() }()
	env.waitConsumed(t)

	select {
	case err := <-result:
		t.Fatalf("Wrong match. expect: still running before the signal, got: returned %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	chSigs <- syscall.SIGTERM
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	case <-time.After(signalWait):
		t.Fatalf("Wrong match. expect: returned after the signal, got: timeout")
	}
}

func Test_runService_failures(t *testing.T) {
	errBoom := errors.New("injected failure")

	tests := []struct {
		name string

		setup func(env *serviceEnv)

		expectErr        string
		expectEngineUp   int32
		expectEngineOpen bool
	}{
		{
			name: "database connection fails",
			setup: func(env *serviceEnv) {
				env.connectErr = errBoom
			},
			expectErr: "injected failure",
		},
		{
			name: "migration engine cannot be opened",
			setup: func(env *serviceEnv) {
				env.engineErr = errBoom
			},
			expectErr: "injected failure",
		},
		{
			name: "migration version cannot be read",
			setup: func(env *serviceEnv) {
				env.engine.versionErr = errBoom
			},
			expectErr:        "could not get the migration version",
			expectEngineOpen: true,
		},
		{
			name: "migration up fails",
			setup: func(env *serviceEnv) {
				env.engine.upErr = errBoom
			},
			expectErr:        "could not apply the migrations",
			expectEngineUp:   1,
			expectEngineOpen: true,
		},
		{
			name: "cache is not reachable",
			setup: func(env *serviceEnv) {
				// the server is closed, so the address refuses the connection without any external state.
				env.redis.Close()
			},
			expectErr:        "connection refused",
			expectEngineUp:   1,
			expectEngineOpen: true,
		},
		{
			name: "queue cannot be created",
			setup: func(env *serviceEnv) {
				env.expectListen(errBoom)
			},
			expectErr:        "could not declare the queue for listenHandler",
			expectEngineUp:   1,
			expectEngineOpen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newServiceEnv(t)
			tt.setup(env)

			err := runServiceWithSignal(t)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong match. expect: error containing %q, got: %v", tt.expectErr, err)
			}

			if got := env.engine.upCalls.Load(); got != tt.expectEngineUp {
				t.Errorf("Wrong migration up count. expect: %d, got: %d", tt.expectEngineUp, got)
			}
			if got := env.engine.closeCalls.Load() == 1; got != tt.expectEngineOpen {
				t.Errorf("Wrong migration close state. expect: %v, got: %v", tt.expectEngineOpen, got)
			}
		})
	}
}

func Test_run_and_runListen(t *testing.T) {
	t.Run("runListen starts the consumer with the configured broker address", func(t *testing.T) {
		env := newServiceEnv(t)
		env.expectListen(nil)

		if err := run(env.db, nil, config.Get()); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		env.waitConsumed(t)

		if got := env.rabbitURI.Load(); got != testRabbitAddress {
			t.Errorf("Wrong broker address. expect: %s, got: %v", testRabbitAddress, got)
		}
	})

	t.Run("run returns the listen error", func(t *testing.T) {
		env := newServiceEnv(t)
		env.expectListen(errors.New("queue failure"))

		err := run(env.db, nil, config.Get())
		if err == nil || !strings.Contains(err.Error(), "queue failure") {
			t.Fatalf("Wrong match. expect: queue failure, got: %v", err)
		}
	})
}

// Test_main_runs_service runs main without arguments. The config loader runs once per process, so the test uses it up
// first and then sets the values; the command line itself carries no flags. The service stops through the queued signal.
func Test_main_runs_service(t *testing.T) {
	env := newServiceEnv(t)
	env.expectListen(nil)

	origArgs := os.Args
	origLevel := logrus.GetLevel()
	origFormatter := logrus.StandardLogger().Formatter
	viper.Reset()
	config.LoadGlobalConfig()
	t.Cleanup(func() {
		os.Args = origArgs
		viper.Reset()
		logrus.SetLevel(origLevel)
		logrus.SetFormatter(origFormatter)
	})

	// LoadGlobalConfig may have replaced the values of newServiceEnv, so they are set again.
	cfg := config.Get()
	cfg.DatabaseDSN = "test-database-dsn"
	cfg.PrometheusEndpoint = ""
	cfg.PrometheusListenAddress = ""
	cfg.RabbitMQAddress = testRabbitAddress
	cfg.RedisAddress = env.redis.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	os.Args = []string{"message-manager"}

	chSigs <- syscall.SIGTERM
	main()
	env.waitConsumed(t)

	if got := env.engine.upCalls.Load(); got != 1 {
		t.Errorf("Wrong migration up count. expect: 1, got: %d", got)
	}
	if got := env.rabbitURI.Load(); got != testRabbitAddress {
		t.Errorf("Wrong broker address. expect: %s, got: %v", testRabbitAddress, got)
	}
}
