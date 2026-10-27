package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/smotes/purse"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-route-manager/internal/config"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

// The tests in this file replace process wide state (the connect function, the migration engine,
// the broker factory, the config fields, the prometheus registerer), so none of them runs in parallel.

const waitTimeout = 5 * time.Second

var (
	endpointCounter atomic.Int64
	dbCounter       atomic.Int64
)

// uniqueEndpoint returns a metrics endpoint that was not registered on the default mux before.
func uniqueEndpoint() string {
	return fmt.Sprintf("/test-metrics-%d", endpointCounter.Add(1))
}

// useNewRegistry replaces the default prometheus registerer for the test, because the request handler
// registers its collectors without a guard and a second registration in one process panics.
func useNewRegistry(t *testing.T) {
	t.Helper()

	orig := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = orig })
}

// drainDone empties the shared done channel when the test ends. A run that fails before it waits for
// the done signal leaves the token in the channel.
func drainDone(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		select {
		case <-chDone:
		default:
		}
	})
}

// newTestDB returns an in-memory sqlite database with the schema of the service.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	name := fmt.Sprintf("file:routemanager%d?mode=memory&cache=shared", dbCounter.Add(1))
	db, err := sql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("could not open the database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ps, err := purse.New(filepath.Join("../../scripts/database_scripts"))
	if err != nil {
		t.Fatalf("could not load the schema. err: %v", err)
	}
	for _, f := range ps.Files() {
		contents, ok := ps.Get(f)
		if !ok {
			t.Fatalf("could not get the schema file. file: %s", f)
		}
		if _, errExec := db.Exec(contents); errExec != nil {
			t.Fatalf("could not apply the schema. file: %s, err: %v", f, errExec)
		}
	}

	return db
}

// fakeEngine is a migration engine that never touches a database.
type fakeEngine struct {
	upErr    error
	upCalls  atomic.Int32
	closeCnt atomic.Int32
}

func (e *fakeEngine) Version() (uint, bool, error) { return 0, false, nil }
func (e *fakeEngine) Up() error {
	e.upCalls.Add(1)
	return e.upErr
}
func (e *fakeEngine) Steps(int) error { return nil }
func (e *fakeEngine) Force(int) error { return nil }
func (e *fakeEngine) Close() (error, error) {
	e.closeCnt.Add(1)
	return nil, nil
}

// setConfig changes the config fields for the test and restores them at the end.
func setConfig(t *testing.T, redisAddr string) {
	t.Helper()

	cfg := config.Get()
	origRedis, origEndpoint, origListen, origInterval := cfg.RedisAddress, cfg.PrometheusEndpoint, cfg.PrometheusListenAddress, cfg.HealthCheckInterval
	t.Cleanup(func() {
		cfg.RedisAddress, cfg.PrometheusEndpoint, cfg.PrometheusListenAddress, cfg.HealthCheckInterval = origRedis, origEndpoint, origListen, origInterval
	})

	cfg.RedisAddress = redisAddr
	cfg.PrometheusEndpoint = uniqueEndpoint()
	cfg.PrometheusListenAddress = "127.0.0.1:0"
	cfg.HealthCheckInterval = time.Hour
}

// seamEnv installs the connect function, the migration engine and the broker factory.
func seamEnv(t *testing.T, engine *fakeEngine, mockSock sockhandler.SockHandler) {
	t.Helper()

	restoreDB := databasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return newTestDB(t), nil })
	t.Cleanup(restoreDB)

	restoreEngine := migrationhandler.SetEngineFunc(func(string, fs.FS, string) (migrationhandler.Engine, uint, error) {
		return engine, 1, nil
	})
	t.Cleanup(restoreEngine)

	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)
}

// expectStart sets the calls the service makes on the broker when it starts, and returns the channel
// that is closed when the request queue consumer is started (it is started from a goroutine).
func expectStart(mockSock *sockhandler.MockSockHandler) <-chan struct{} {
	consuming := make(chan struct{})

	mockSock.EXPECT().Connect().Times(1)
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockSock.EXPECT().QueueCreate(gomock.Any(), "normal").Return(nil).Times(1)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), "route-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgRPC) error {
			close(consuming)
			return nil
		}).Times(1)

	return consuming
}

// startRun calls run in a goroutine and returns the channel of its result.
func startRun() <-chan error {
	res := make(chan error, 1)
	go func() { res <- run(rootCmd, nil) }()
	return res
}

// waitRun waits for the result of run and fails the test when it does not return.
func waitRun(t *testing.T, res <-chan error) error {
	t.Helper()

	select {
	case err := <-res:
		return err
	case <-time.After(waitTimeout):
		t.Fatalf("Timed out. waiting for run to return")
		return nil
	}
}

// runWithTimeout calls run and fails the test when it does not return.
func runWithTimeout(t *testing.T) error {
	t.Helper()

	return waitRun(t, startRun())
}

func Test_run_start(t *testing.T) {
	useNewRegistry(t)
	drainDone(t)

	mr := miniredis.RunT(t)
	setConfig(t, mr.Addr())

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	consuming := expectStart(mockSock)

	engine := &fakeEngine{}
	seamEnv(t, engine, mockSock)

	res := startRun()

	select {
	case <-consuming:
	case <-time.After(waitTimeout):
		t.Fatalf("Timed out. waiting for ConsumeRPC")
	}

	// the service keeps running until the done signal arrives
	select {
	case err := <-res:
		t.Fatalf("Wrong match. expect: run keeps running, got: returned with %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	chDone <- true
	if err := waitRun(t, res); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if engine.upCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: 1 migration run, got: %d", engine.upCalls.Load())
	}
	if engine.closeCnt.Load() != 1 {
		t.Errorf("Wrong match. expect: engine closed once, got: %d", engine.closeCnt.Load())
	}
}

func Test_run_migrationFailure(t *testing.T) {
	useNewRegistry(t)
	drainDone(t)

	mr := miniredis.RunT(t)
	setConfig(t, mr.Addr())

	// the broker has no expectation: a call before the migration finished would fail the test
	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	errUp := errors.New("migration failed")
	engine := &fakeEngine{upErr: errUp}
	seamEnv(t, engine, mockSock)

	err := runWithTimeout(t)
	if err == nil || !strings.Contains(err.Error(), errUp.Error()) {
		t.Errorf("Wrong match. expect: %v, got: %v", errUp, err)
	}
	if engine.upCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: 1 migration run, got: %d", engine.upCalls.Load())
	}
}

func Test_run_cacheFailure(t *testing.T) {
	useNewRegistry(t)
	drainDone(t)

	// an address that was served and is closed now refuses the connection
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close()
	setConfig(t, addr)

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)

	engine := &fakeEngine{}
	seamEnv(t, engine, mockSock)

	err := runWithTimeout(t)
	var opErr *net.OpError
	if !errors.As(err, &opErr) || opErr.Op != "dial" {
		t.Fatalf("Wrong match. expect: a dial error of the cache connection, got: %v", err)
	}
	if engine.upCalls.Load() != 1 {
		t.Errorf("Wrong match. expect: the migration ran before the cache, got: %d runs", engine.upCalls.Load())
	}
}

func Test_runService(t *testing.T) {
	useNewRegistry(t)

	mr := miniredis.RunT(t)
	setConfig(t, mr.Addr())

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockSock := sockhandler.NewMockSockHandler(mc)
	consuming := expectStart(mockSock)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)

	// the context stops the health check goroutine
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	if err := runService(ctx, newTestDB(t), mockCache); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-consuming:
	case <-time.After(waitTimeout):
		t.Fatalf("Timed out. waiting for ConsumeRPC")
	}
}
