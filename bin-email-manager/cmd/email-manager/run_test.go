package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/internal/config"
	"monorepo/bin-email-manager/migrations"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"
)

// fakeEngine is a migration engine that touches no database.
type fakeEngine struct {
	upErr error
	ups   int
	close int
}

func (f *fakeEngine) Version() (uint, bool, error) { return 0, false, nil }
func (f *fakeEngine) Up() error                    { f.ups++; return f.upErr }
func (f *fakeEngine) Steps(int) error              { return nil }
func (f *fakeEngine) Force(int) error              { return nil }
func (f *fakeEngine) Close() (error, error)        { f.close++; return nil, nil }

// isolateMetrics gives the test its own prometheus registry, because requesthandler registers
// its collectors on the default registerer on every construction and a repeat would panic.
func isolateMetrics(t *testing.T) {
	t.Helper()

	saved := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = saved })
}

// useConfig overwrites the global config for the test and restores it afterwards.
func useConfig(t *testing.T, mod func(cfg *config.Config)) *config.Config {
	t.Helper()

	cfg := config.Get()
	saved := *cfg
	t.Cleanup(func() { *cfg = saved })

	*cfg = config.Config{}
	mod(cfg)

	return cfg
}

// seamRecord keeps what the fake backends were asked for.
type seamRecord struct {
	connectDSN string
	engineDSN  string
	engineFS   fs.FS
	engineTbl  string
	rabbitURI  string

	engine *fakeEngine
}

// useFakeBackends replaces the database connection, the migration engine and the RabbitMQ
// connection with fakes and returns what they were asked for. These seams are process-global,
// so the callers must not run in parallel.
func useFakeBackends(t *testing.T, connectErr error, engine *fakeEngine, engineErr error, sockHandler sockhandler.SockHandler) *seamRecord {
	t.Helper()

	rec := &seamRecord{engine: engine}

	restoreConnect := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		rec.connectDSN = dsn
		if connectErr != nil {
			return nil, connectErr
		}

		db, _, err := sqlmock.New()
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() { _ = db.Close() })

		return db, nil
	})
	t.Cleanup(restoreConnect)

	restoreEngine := migrationhandler.SetEngineFunc(func(dsn string, fsys fs.FS, table string) (migrationhandler.Engine, uint, error) {
		rec.engineDSN = dsn
		rec.engineFS = fsys
		rec.engineTbl = table
		if engineErr != nil {
			return nil, 0, engineErr
		}

		return engine, 1, nil
	})
	t.Cleanup(restoreEngine)

	restoreRabbit := sockhandler.SetRabbitFactory(func(uri string) sockhandler.SockHandler {
		rec.rabbitURI = uri
		return sockHandler
	})
	t.Cleanup(restoreRabbit)

	return rec
}

func Test_createDBHandler(t *testing.T) {
	tests := []struct {
		name string

		connectErr error
		engineErr  error
		upErr      error
		cacheDown  bool

		expectErr    string
		expectEngine bool
		expectHandle bool
	}{
		{
			name:         "success",
			expectEngine: true,
			expectHandle: true,
		},
		{
			name:       "connect_fails_before_migration",
			connectErr: fmt.Errorf("connect refused"),
			expectErr:  "connect refused",
		},
		{
			name:         "migration_open_fails",
			engineErr:    fmt.Errorf("engine open failed"),
			expectErr:    "engine open failed",
			expectEngine: false,
		},
		{
			name:         "migration_up_fails",
			upErr:        fmt.Errorf("up failed"),
			expectErr:    "could not apply the migrations: up failed",
			expectEngine: true,
		},
		{
			name:         "cache_connect_fails",
			cacheDown:    true,
			expectErr:    "connection refused",
			expectEngine: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			redisAddr := mr.Addr()
			if tt.cacheDown {
				redisAddr = closedLoopbackAddr(t)
			}

			cfg := useConfig(t, func(cfg *config.Config) {
				cfg.DatabaseDSN = "synthetic-dsn"
				cfg.RedisAddress = redisAddr
			})

			engine := &fakeEngine{upErr: tt.upErr}
			rec := useFakeBackends(t, tt.connectErr, engine, tt.engineErr, nil)

			res, err := createDBHandler(cfg)

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
				}
			} else if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectHandle != (res != nil) {
				t.Errorf("Wrong handler. expect handler: %v, got: %v", tt.expectHandle, res)
			}

			if rec.connectDSN != "synthetic-dsn" {
				t.Errorf("Wrong connect dsn. expect: synthetic-dsn, got: %q", rec.connectDSN)
			}

			if tt.expectEngine {
				if rec.engineDSN != "synthetic-dsn" || rec.engineTbl != migrations.VersionTable {
					t.Errorf("Wrong migration input. dsn: %q, table: %q", rec.engineDSN, rec.engineTbl)
				}
				if engine.ups != 1 || engine.close != 1 {
					t.Errorf("Wrong engine usage. expect up 1 close 1, got up %d close %d", engine.ups, engine.close)
				}
			} else if rec.engineDSN != "" && tt.connectErr != nil {
				t.Errorf("Expected no migration after a connection failure. got dsn: %q", rec.engineDSN)
			}
		})
	}
}

// expectListen sets the sock expectations for a service that connects, declares the global
// topic and starts listening. The returned channel is closed when ConsumeRPC is called.
func expectListen(mockSock *sockhandler.MockSockHandler) <-chan struct{} {
	consumed := make(chan struct{})

	mockSock.EXPECT().Connect()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil)
	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameEmailRequest), "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), string(commonoutline.QueueNameEmailRequest), gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ any, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			close(consumed)
			return nil
		})

	return consumed
}

func Test_run(t *testing.T) {
	isolateMetrics(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	consumed := expectListen(mockSock)

	cfg := useConfig(t, func(cfg *config.Config) {
		cfg.RabbitMQAddress = "synthetic-broker"
	})
	rec := useFakeBackends(t, nil, &fakeEngine{}, nil, mockSock)

	run(mockDB, cfg)

	select {
	case <-consumed:
	case <-time.After(2 * time.Second):
		t.Fatalf("The request queue was not consumed")
	}

	if rec.rabbitURI != "synthetic-broker" {
		t.Errorf("Wrong broker address. expect: synthetic-broker, got: %q", rec.rabbitURI)
	}
}

func Test_run_queueCreateFails(t *testing.T) {
	isolateMetrics(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockSock.EXPECT().Connect()
	mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), gomock.Any()).Return(nil)
	// ConsumeRPC has no expectation: a failed queue declaration must not start consuming
	mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameEmailRequest), "normal").Return(fmt.Errorf("queue failed"))

	cfg := useConfig(t, func(cfg *config.Config) {})
	useFakeBackends(t, nil, &fakeEngine{}, nil, mockSock)

	// the failure is logged and run returns
	run(mockDB, cfg)
}

// drainDone empties the done channel that an earlier test may have left filled.
func drainDone() {
	select {
	case <-chDone:
	default:
	}
}

func Test_runService(t *testing.T) {
	isolateMetrics(t)
	drainDone()
	t.Cleanup(func() {
		signal.Reset(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
		drainDone()
	})

	mr := miniredis.RunT(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	consumed := expectListen(mockSock)

	useConfig(t, func(cfg *config.Config) {
		// the prometheus server stays disabled
		cfg.DatabaseDSN = "synthetic-dsn"
		cfg.RedisAddress = mr.Addr()
		cfg.RabbitMQAddress = "synthetic-broker"
	})
	engine := &fakeEngine{}
	useFakeBackends(t, nil, engine, nil, mockSock)

	result := make(chan error, 1)
	go func() { result <- runService() }()

	select {
	case <-consumed:
	case <-time.After(5 * time.Second):
		t.Fatalf("The service did not start listening")
	}

	// the service keeps running until a signal arrives
	select {
	case err := <-result:
		t.Fatalf("The service stopped before a signal. err: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	chSigs <- syscall.SIGTERM

	select {
	case err := <-result:
		if err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("The service did not stop after the signal")
	}

	if engine.ups != 1 {
		t.Errorf("Expected the migrations to run once. got: %d", engine.ups)
	}
}

func Test_runService_migrationFails(t *testing.T) {
	isolateMetrics(t)
	t.Cleanup(func() { signal.Reset(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM) })

	useConfig(t, func(cfg *config.Config) {
		cfg.DatabaseDSN = "synthetic-dsn"
	})
	useFakeBackends(t, nil, &fakeEngine{}, fmt.Errorf("engine open failed"), nil)

	err := runService()
	if err == nil || !strings.Contains(err.Error(), "engine open failed") {
		t.Errorf("Wrong match. expect the migration error, got: %v", err)
	}
}
