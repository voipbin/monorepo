package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/pkg/cachehandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

var dbSeq atomic.Int64

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

// testEnv holds the in-process stand-ins that replace the database, the cache and the broker.
type testEnv struct {
	db    *sql.DB
	redis *miniredis.Miniredis
	sock  *sockhandler.MockSockHandler
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

// newTestEnv installs the factories of the shared library and points the configuration at the
// stand-ins. Everything is restored when the test ends. The test must not run in parallel.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	// resets viper and restores the configuration when the test ends.
	setupTest(t)

	// the once-only load must not overwrite the values assigned below, whatever the test order is.
	config.LoadGlobalConfig()

	// the request handler registers its metrics on the default registerer at every construction and
	// panics on the second one, so each test gets a registry that accepts the repeated registration.
	prevRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = &tolerantRegisterer{Registry: prometheus.NewRegistry()}
	t.Cleanup(func() { prometheus.DefaultRegisterer = prevRegisterer })

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
	t.Setenv("STORAGE_EMULATOR_HOST", "http://"+closedLoopbackAddr(t))

	redisServer := miniredis.RunT(t)
	cfg := config.Get()
	cfg.RedisAddress = redisServer.Addr()
	cfg.RedisPassword = ""
	cfg.RedisDatabase = 0
	cfg.RabbitMQAddress = "unused"
	cfg.GCPProjectID = "test-project"
	cfg.GCPBucketNameMedia = "test-media"
	cfg.GCPBucketNameTmp = "test-tmp"

	db, err := sql.Open("sqlite3", fmt.Sprintf("file:ctltest%d?mode=memory&cache=shared", dbSeq.Add(1)))
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

	mc := gomock.NewController(t)
	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().Connect().AnyTimes()
	mockSock.EXPECT().TopicCreate(gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockSock.EXPECT().EventPublish(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return mockSock })
	t.Cleanup(restoreSock)

	return &testEnv{db: db, redis: redisServer, sock: mockSock}
}

// seedHandler returns a database handler on the stand-ins, used to create the fixtures.
func (e *testEnv) seedHandler(t *testing.T) dbhandler.DBHandler {
	t.Helper()

	cache := cachehandler.NewHandler(e.redis.Addr(), "", 0)
	if err := cache.Connect(); err != nil {
		t.Fatalf("could not connect to the cache. err: %v", err)
	}

	return dbhandler.NewHandler(e.db, cache)
}
