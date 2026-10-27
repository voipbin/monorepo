package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-storage-manager/internal/config"
)

const testAccountID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

var (
	dbSeq       atomic.Int64
	endpointSeq atomic.Int64
)

// tolerantRegisterer ignores the error of registering a collector that is already registered.
type tolerantRegisterer struct {
	*prometheus.Registry
}

func (r *tolerantRegisterer) MustRegister(cs ...prometheus.Collector) {
	for _, c := range cs {
		if err := r.Register(c); err != nil {
			var already prometheus.AlreadyRegisteredError
			if !errors.As(err, &already) {
				panic(err)
			}
		}
	}
}

// fakeEngine is a migration engine that touches nothing.
type fakeEngine struct {
	upErr    error
	upCalls  int
	closed   int
	gotDSN   string
	gotTable string
	gotFS    fs.FS
}

func (e *fakeEngine) Version() (uint, bool, error) { return 0, false, nil }
func (e *fakeEngine) Up() error {
	e.upCalls++
	return e.upErr
}
func (e *fakeEngine) Steps(int) error { return nil }
func (e *fakeEngine) Force(int) error { return nil }
func (e *fakeEngine) Close() (error, error) {
	e.closed++
	return nil, nil
}

// testEnv holds the in-process stand-ins that replace the database, the cache, the broker and the
// migration engine.
type testEnv struct {
	db     *sql.DB
	redis  *miniredis.Miniredis
	sock   *sockhandler.MockSockHandler
	engine *fakeEngine
}

// closedLoopbackAddr returns a loopback address that nothing listens on.
func closedLoopbackAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not reserve a loopback address. err: %v", err)
	}
	addr := l.Addr().String()
	if errClose := l.Close(); errClose != nil {
		t.Fatalf("could not release the loopback address. err: %v", errClose)
	}

	return addr
}

// uniqueEndpoint returns a metrics endpoint that was not registered on the default mux before.
func uniqueEndpoint() string {
	return fmt.Sprintf("/metrics-test-%d", endpointSeq.Add(1))
}

// newTestEnv installs the factories of the shared library and points the configuration at the
// stand-ins. Everything is restored when the test ends. The test must not run in parallel.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	// the once-only load must not overwrite the values assigned below, whatever the test order is.
	config.LoadGlobalConfig()
	old := *config.Get()
	t.Cleanup(func() { *config.Get() = old })

	// the request handler registers its metrics on the default registerer at every construction and
	// panics on the second one, so each test gets a registry that accepts the repeated registration.
	prevRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = &tolerantRegisterer{Registry: prometheus.NewRegistry()}
	t.Cleanup(func() { prometheus.DefaultRegisterer = prevRegisterer })

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("STORAGE_EMULATOR_HOST", "http://"+closedLoopbackAddr(t))

	redisServer := miniredis.RunT(t)
	cfg := config.Get()
	cfg.DatabaseDSN = "test-dsn"
	cfg.RedisAddress = redisServer.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	cfg.RabbitMQAddress = "unused"
	cfg.GCPProjectID = "test-project"
	cfg.GCPBucketNameMedia = "test-media"
	cfg.GCPBucketNameTmp = "test-tmp"
	cfg.PrometheusEndpoint = uniqueEndpoint()
	cfg.PrometheusListenAddress = closedLoopbackAddr(t)

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:mgrtest%d?mode=memory&cache=shared", dbSeq.Add(1)))
	if err != nil {
		t.Fatalf("could not open the sqlite database. err: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	dir := filepath.Join("..", "..", "scripts", "database_scripts_test")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("could not read the schema scripts. err: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		content, errRead := os.ReadFile(filepath.Join(dir, name)) //nolint:gosec // fixed test schema directory
		if errRead != nil {
			t.Fatalf("could not read the schema script. err: %v", errRead)
		}
		if _, errExec := db.Exec(string(content)); errExec != nil {
			t.Fatalf("could not apply the schema script. name: %s, err: %v", name, errExec)
		}
	}

	restoreConnect := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return db, nil })
	t.Cleanup(restoreConnect)

	engine := &fakeEngine{}
	restoreEngine := migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, versionTable string) (migrationhandler.Engine, uint, error) {
		engine.gotDSN = dsn
		engine.gotTable = versionTable
		engine.gotFS = fsys
		return engine, 1, nil
	})
	t.Cleanup(restoreEngine)

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().AnyTimes()
	mockSock.EXPECT().TopicCreate(gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)

	return &testEnv{db: db, redis: redisServer, sock: mockSock, engine: engine}
}
