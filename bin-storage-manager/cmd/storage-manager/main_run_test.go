package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"

	"monorepo/bin-storage-manager/internal/config"
	"monorepo/bin-storage-manager/migrations"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

const pollTimeout = 15 * time.Second

// expectConsumers allows the two queue creations and consumers of a successful start and returns a
// channel that receives one value per started consumer.
func expectConsumers(env *testEnv) <-chan string {
	started := make(chan string, 2)

	reqQueue := string(commonoutline.QueueNameStorageRequest)
	subQueue := string(commonoutline.QueueNameStorageSubscribe)

	env.sock.EXPECT().QueueCreate(reqQueue, "normal").Return(nil)
	env.sock.EXPECT().ConsumeRPC(gomock.Any(), reqQueue, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
			started <- queueName
			return nil
		})
	env.sock.EXPECT().QueueCreate(subQueue, "normal").Return(nil)
	env.sock.EXPECT().QueueBind(subQueue, gomock.Any(), string(commonoutline.QueueNameEvent), false, nil).Return(nil).MinTimes(1)
	env.sock.EXPECT().ConsumeMessage(gomock.Any(), subQueue, gomock.Any(), false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, numWorkers int, cb sock.CbMsgConsume) error {
			started <- queueName
			return nil
		})

	return started
}

func waitConsumers(t *testing.T, started <-chan string) {
	t.Helper()

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case name := <-started:
			got[name] = true
		case <-time.After(waitTimeout):
			t.Fatalf("Wrong match. a consumer was not started. started: %v", got)
		}
	}

	if !got[string(commonoutline.QueueNameStorageRequest)] || !got[string(commonoutline.QueueNameStorageSubscribe)] {
		t.Errorf("Wrong match. expect both queues to be consumed, got: %v", got)
	}
}

func Test_createDBHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		env := newTestEnv(t)

		res, err := createDBHandler()
		if err != nil || res == nil {
			t.Fatalf("Wrong match. expect: ok, got: %v, %v", res, err)
		}

		// the migration ran once on the configured dsn and version table, and was closed
		if env.engine.upCalls != 1 || env.engine.closed != 1 {
			t.Errorf("Wrong match. expect one Up and one Close, got: %d and %d", env.engine.upCalls, env.engine.closed)
		}
		if env.engine.gotDSN != "test-dsn" || env.engine.gotTable != migrations.VersionTable || env.engine.gotFS == nil {
			t.Errorf("Wrong match. dsn: %s, table: %s, fs: %v", env.engine.gotDSN, env.engine.gotTable, env.engine.gotFS)
		}

		// the returned handler works on the connected database
		_, errGet := res.AccountGet(context.Background(), uuid.FromStringOrNil(testAccountID))
		if !errors.Is(errGet, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect: %v, got: %v", dbhandler.ErrNotFound, errGet)
		}
	})

	t.Run("connect error stops before the migration", func(t *testing.T) {
		env := newTestEnv(t)
		restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return nil, errSimulated })
		defer restore()

		res, err := createDBHandler()
		if !errors.Is(err, errSimulated) || res != nil {
			t.Errorf("Wrong match. expect: %v and nil handler, got: %v, %v", errSimulated, err, res)
		}
		if env.engine.upCalls != 0 {
			t.Errorf("Wrong match. expect no migration, got: %d", env.engine.upCalls)
		}
	})

	t.Run("migration engine open error", func(t *testing.T) {
		newTestEnv(t)
		restore := migrationhandler.SetEngineFunc(func(string, fs.FS, string) (migrationhandler.Engine, uint, error) {
			return nil, 0, errSimulated
		})
		defer restore()

		res, err := createDBHandler()
		if !errors.Is(err, errSimulated) || res != nil {
			t.Errorf("Wrong match. expect: %v and nil handler, got: %v, %v", errSimulated, err, res)
		}
	})

	t.Run("migration error", func(t *testing.T) {
		env := newTestEnv(t)
		env.engine.upErr = errSimulated

		res, err := createDBHandler()
		if !errors.Is(err, errSimulated) || res != nil {
			t.Errorf("Wrong match. expect: %v and nil handler, got: %v, %v", errSimulated, err, res)
		}
		if env.engine.closed != 1 {
			t.Errorf("Wrong match. expect the engine to be closed, got: %d", env.engine.closed)
		}
	})

	t.Run("cache down", func(t *testing.T) {
		env := newTestEnv(t)
		env.redis.Close()

		res, err := createDBHandler()
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: error and nil handler, got: %v, %v", err, res)
		}
		if env.engine.upCalls != 1 {
			t.Errorf("Wrong match. the migration runs before the cache. expect: 1, got: %d", env.engine.upCalls)
		}
	})
}

func Test_runService(t *testing.T) {
	t.Run("both consumers started", func(t *testing.T) {
		env := newTestEnv(t)
		started := expectConsumers(env)

		dbHandler, err := createDBHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if err := runService(dbHandler); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		waitConsumers(t, started)
	})

	t.Run("unreadable credential file", func(t *testing.T) {
		newTestEnv(t)
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", t.TempDir()+"/missing.json")

		dbHandler, err := createDBHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		// no queue is declared, because the mock has no expectation for it
		err = runService(dbHandler)
		if err == nil || !strings.Contains(err.Error(), "failed to read credential file") {
			t.Errorf("Wrong match. expect: failed to read credential file, got: %v", err)
		}
	})

	t.Run("cache down", func(t *testing.T) {
		env := newTestEnv(t)

		dbHandler, err := createDBHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		env.redis.Close()

		// no queue is declared, because the mock has no expectation for it
		if err := runService(dbHandler); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("listener queue error", func(t *testing.T) {
		env := newTestEnv(t)
		env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameStorageRequest), "normal").Return(errSimulated)

		dbHandler, err := createDBHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		// the subscriber queue is not declared, because the mock has no expectation for it
		if err := runService(dbHandler); !errors.Is(err, errSimulated) {
			t.Errorf("Wrong match. expect: %v, got: %v", errSimulated, err)
		}
	})

	t.Run("subscriber queue error", func(t *testing.T) {
		env := newTestEnv(t)
		started := make(chan struct{}, 1)
		reqQueue := string(commonoutline.QueueNameStorageRequest)
		env.sock.EXPECT().QueueCreate(reqQueue, "normal").Return(nil)
		env.sock.EXPECT().ConsumeRPC(gomock.Any(), reqQueue, gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
				started <- struct{}{}
				return nil
			})
		env.sock.EXPECT().QueueCreate(string(commonoutline.QueueNameStorageSubscribe), "normal").Return(errSimulated)

		dbHandler, err := createDBHandler()
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		if err := runService(dbHandler); !errors.Is(err, errSimulated) {
			t.Errorf("Wrong match. expect: %v, got: %v", errSimulated, err)
		}
		select {
		case <-started:
		case <-time.After(waitTimeout):
			t.Fatalf("Wrong match. the listener consumer was not started")
		}
	})
}

// drainDone empties the done channel, so that a test does not leave a value for the next one.
func drainDone(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		select {
		case <-chDone:
		default:
		}
	})
}

func Test_run(t *testing.T) {
	t.Run("returns after the done signal", func(t *testing.T) {
		env := newTestEnv(t)
		drainDone(t)
		started := expectConsumers(env)

		chDone <- true
		if err := run(&cobra.Command{Use: "test"}, nil); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		waitConsumers(t, started)

		if len(chDone) != 0 {
			t.Errorf("Wrong match. expect the done signal to be consumed, got: %d", len(chDone))
		}
		if env.engine.upCalls != 1 {
			t.Errorf("Wrong match. expect one migration, got: %d", env.engine.upCalls)
		}

		// the metrics endpoint of the configuration is served
		pollGet(t, "http://"+config.Get().PrometheusListenAddress+config.Get().PrometheusEndpoint)
	})

	t.Run("database error is returned and does not wait for the signal", func(t *testing.T) {
		newTestEnv(t)
		drainDone(t)
		restore := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return nil, errSimulated })
		defer restore()

		if err := run(&cobra.Command{Use: "test"}, nil); !errors.Is(err, errSimulated) {
			t.Errorf("Wrong match. expect: %v, got: %v", errSimulated, err)
		}
	})

	t.Run("service error is returned and does not wait for the signal", func(t *testing.T) {
		newTestEnv(t)
		drainDone(t)
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", t.TempDir()+"/missing.json")

		err := run(&cobra.Command{Use: "test"}, nil)
		if err == nil || !strings.Contains(err.Error(), "failed to read credential file") {
			t.Errorf("Wrong match. expect: failed to read credential file, got: %v", err)
		}
	})
}

// pollGet requests the url until it answers with 200 and returns the body.
func pollGet(t *testing.T, url string) string {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for {
		res, err := http.Get(url) //nolint:gosec,noctx // loopback address of the test
		if err == nil {
			body, errRead := io.ReadAll(res.Body)
			_ = res.Body.Close()
			if errRead == nil && res.StatusCode == http.StatusOK {
				return string(body)
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("Wrong match. %s did not answer 200 in time. last err: %v", url, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func Test_initProm(t *testing.T) {
	t.Run("serves the endpoint", func(t *testing.T) {
		addr := closedLoopbackAddr(t)
		endpoint := uniqueEndpoint()

		initProm(endpoint, addr)

		body := pollGet(t, "http://"+addr+endpoint)
		if !strings.Contains(body, "go_goroutines") {
			t.Errorf("Wrong match. expect the go runtime metrics in the body")
		}
	})

	t.Run("retries until the address is free", func(t *testing.T) {
		hook := logrustest.NewGlobal()
		t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks)) })

		holder, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not hold a loopback address. err: %v", err)
		}
		addr := holder.Addr().String()
		endpoint := uniqueEndpoint()

		initProm(endpoint, addr)

		// the first attempt fails while the address is held
		deadline := time.Now().Add(pollTimeout)
		for {
			failed := false
			for _, e := range hook.AllEntries() {
				if strings.Contains(e.Message, "Could not start prometheus listener") {
					failed = true
				}
			}
			if failed {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("Wrong match. the failed attempt was not logged")
			}
			time.Sleep(10 * time.Millisecond)
		}

		if errClose := holder.Close(); errClose != nil {
			t.Fatalf("could not release the address. err: %v", errClose)
		}

		pollGet(t, "http://"+addr+endpoint)
	})
}
