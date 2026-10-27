package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/migrationhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/internal/config"
	"monorepo/bin-direct-manager/pkg/cachehandler"
)

var errEngineTest = errors.New("engine failure")

// fakeEngine records the calls of the migration runner.
type fakeEngine struct {
	upErr error

	ups    atomic.Int32
	closes atomic.Int32
}

func (f *fakeEngine) Version() (uint, bool, error) { return 0, false, nil }
func (f *fakeEngine) Up() error                    { f.ups.Add(1); return f.upErr }
func (f *fakeEngine) Steps(int) error              { return nil }
func (f *fakeEngine) Force(int) error              { return nil }
func (f *fakeEngine) Close() (error, error)        { f.closes.Add(1); return nil, nil }

// fakeBroker is a mock broker that reports when each consumer is started.
type fakeBroker struct {
	sock *sockhandler.MockSockHandler

	connects     atomic.Int32
	rpcStarted   chan struct{}
	eventStarted chan struct{}
}

// useFakeBroker replaces the broker factory and gives every test its own metrics registry
// (the request handler and the circuit breaker register collectors on every construction).
// subscribeQueueErr is returned by the creation of the event queue.
func useFakeBroker(t *testing.T, subscribeQueueErr error) *fakeBroker {
	t.Helper()

	prevRegisterer := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = prevRegisterer })

	mc := gomock.NewController(t)
	b := &fakeBroker{
		sock:         sockhandler.NewMockSockHandler(mc),
		rpcStarted:   make(chan struct{}),
		eventStarted: make(chan struct{}),
	}
	requestQueue := string(commonoutline.QueueNameDirectRequest)
	eventQueue := string(commonoutline.QueueNameDirectSubscribe)
	exchange := string(commonoutline.QueueNameEvent)

	b.sock.EXPECT().Connect().Do(func() { b.connects.Add(1) }).AnyTimes()
	// a failure of this call ends the process
	b.sock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil).AnyTimes()
	b.sock.EXPECT().QueueCreate(requestQueue, "normal").Return(nil).AnyTimes()
	b.sock.EXPECT().QueueCreate(eventQueue, "normal").Return(subscribeQueueErr).AnyTimes()
	b.sock.EXPECT().QueueBind(eventQueue, "customer-manager.customer.*.deleted", exchange, false, nil).Return(nil).AnyTimes()
	b.sock.EXPECT().ConsumeRPC(gomock.Any(), requestQueue, "direct-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			close(b.rpcStarted)
			return nil
		}).AnyTimes()
	b.sock.EXPECT().ConsumeMessage(gomock.Any(), eventQueue, "direct-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgConsume) error {
			close(b.eventStarted)
			return nil
		}).AnyTimes()

	t.Cleanup(sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler { return b.sock }))

	return b
}

// waitStarted waits, with an upper bound, for a consumer to be started.
func waitStarted(t *testing.T, started chan struct{}, what string) {
	t.Helper()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatalf("the %s consumer was not started", what)
	}
}

// notStarted reports whether a consumer has not been started.
func notStarted(started chan struct{}) bool {
	select {
	case <-started:
		return false
	default:
		return true
	}
}

// useFakeEngine replaces the migration engine; the engine reports no version and a maximum
// version of 1, so Up runs.
func useFakeEngine(t *testing.T, e migrationhandler.Engine) {
	t.Helper()

	t.Cleanup(migrationhandler.SetEngineFunc(func(string, fs.FS, string) (migrationhandler.Engine, uint, error) {
		return e, 1, nil
	}))
}

// useSQLMock returns a database that expects to be closed once, and makes the connection
// seam return it.
func useSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	t.Cleanup(commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) { return db, nil }))

	return db, mock
}

// drainDone empties chDone without blocking, so a test that returns before the receive of run
// does not leave a value for the next test.
func drainDone(t *testing.T) {
	t.Helper()

	t.Cleanup(func() {
		select {
		case <-chDone:
		default:
		}
	})
}

// newRunCommand returns a command whose flags make run use the given cache address.
func newRunCommand(t *testing.T, redisAddress string) *cobra.Command {
	t.Helper()

	cmd := newServiceCommand(t, "unused", uniqueEndpoint(), "127.0.0.1:0")
	if err := cmd.Flags().Set("redis_address", redisAddress); err != nil {
		t.Fatalf("could not set the flag: %v", err)
	}

	return cmd
}

// runBounded runs run in a goroutine and fails the test instead of hanging when it does not
// return, which is what a removed failure branch looks like (run then waits for the done signal).
func runBounded(t *testing.T, cmd *cobra.Command) error {
	t.Helper()

	finished := make(chan error, 1)
	go func() { finished <- run(cmd, nil) }()

	select {
	case err := <-finished:
		return err
	case <-time.After(10 * time.Second):
		t.Fatalf("run() did not return; a failure branch is missing or a step blocks")
		return nil
	}
}

func Test_run_afterDatabase(t *testing.T) {
	t.Run("a migration failure is returned and the database is closed", func(t *testing.T) {
		resetState(t)
		drainDone(t)
		b := useFakeBroker(t, nil)
		engine := &fakeEngine{upErr: errEngineTest}
		useFakeEngine(t, engine)
		_, mock := useSQLMock(t)
		mock.ExpectClose()
		mr := miniredis.RunT(t)

		cmd := newRunCommand(t, mr.Addr())

		err := runBounded(t, cmd)
		if !errors.Is(err, errEngineTest) || !strings.Contains(err.Error(), "could not apply the migrations") {
			t.Errorf("run() error = %v, want the migration failure", err)
		}
		if got := engine.closes.Load(); got != 1 {
			t.Errorf("engine closes = %d, want 1", got)
		}
		if got := b.connects.Load(); got != 0 {
			t.Errorf("the broker was connected %d times, want 0 (the migration comes first)", got)
		}
		if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
			t.Errorf("the database was not closed: %v", errExpect)
		}
	})

	t.Run("a cache failure is returned after the migrations ran", func(t *testing.T) {
		resetState(t)
		drainDone(t)
		b := useFakeBroker(t, nil)
		engine := &fakeEngine{}
		useFakeEngine(t, engine)
		_, mock := useSQLMock(t)
		mock.ExpectClose()
		mr := miniredis.RunT(t)
		addr := mr.Addr()
		mr.Close()

		cmd := newRunCommand(t, addr)

		if err := runBounded(t, cmd); err == nil {
			t.Fatalf("run() error = nil, want the cache connect failure")
		}
		if got := engine.ups.Load(); got != 1 {
			t.Errorf("engine Up calls = %d, want 1", got)
		}
		if got := b.connects.Load(); got != 0 {
			t.Errorf("the broker was connected %d times, want 0", got)
		}
		if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
			t.Errorf("the database was not closed: %v", errExpect)
		}
	})

	t.Run("a subscribe failure of the service is returned", func(t *testing.T) {
		resetState(t)
		drainDone(t)
		b := useFakeBroker(t, errBrokerTest)
		useFakeEngine(t, &fakeEngine{})
		_, mock := useSQLMock(t)
		mock.ExpectClose()
		mr := miniredis.RunT(t)

		cmd := newRunCommand(t, mr.Addr())

		if err := runBounded(t, cmd); !errors.Is(err, errBrokerTest) {
			t.Errorf("run() error = %v, want %v wrapped", err, errBrokerTest)
		}
		waitStarted(t, b.rpcStarted, "request")
		if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
			t.Errorf("the database was not closed: %v", errExpect)
		}
	})

	t.Run("a started service waits for the done signal and then returns", func(t *testing.T) {
		resetState(t)
		drainDone(t)
		b := useFakeBroker(t, nil)
		engine := &fakeEngine{}
		useFakeEngine(t, engine)
		_, mock := useSQLMock(t)
		mock.ExpectClose()
		mr := miniredis.RunT(t)
		// the cache server requires the password given by the flag, so a run that does not pass
		// it on cannot start; the value is synthetic and built at run time
		cachePassword := strings.Repeat("p", 12)
		mr.RequireAuth(cachePassword)

		cmd := newRunCommand(t, mr.Addr())
		if err := cmd.Flags().Set("redis_password", cachePassword); err != nil {
			t.Fatalf("could not set the flag: %v", err)
		}

		finished := make(chan error, 1)
		go func() { finished <- run(cmd, nil) }()

		waitStarted(t, b.rpcStarted, "request")
		waitStarted(t, b.eventStarted, "event")
		select {
		case err := <-finished:
			t.Fatalf("run() returned %v before the done signal", err)
		case <-time.After(100 * time.Millisecond):
		}

		chDone <- true
		select {
		case err := <-finished:
			if err != nil {
				t.Errorf("run() error = %v, want nil", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("run() did not return after the done signal")
		}

		if engine.ups.Load() != 1 || engine.closes.Load() != 1 {
			t.Errorf("engine ups = %d, closes = %d, want 1 and 1", engine.ups.Load(), engine.closes.Load())
		}
		if got := b.connects.Load(); got != 1 {
			t.Errorf("the broker was connected %d times, want 1", got)
		}
		if errExpect := mock.ExpectationsWereMet(); errExpect != nil {
			t.Errorf("the database was not closed: %v", errExpect)
		}
	})
}

func Test_runService(t *testing.T) {
	t.Run("starts both consumers", func(t *testing.T) {
		resetState(t)
		*config.Get() = config.Config{RabbitMQAddress: "unused"}
		b := useFakeBroker(t, nil)
		db, _ := useSQLMock(t)
		mr := miniredis.RunT(t)
		cache := cachehandler.NewHandler(mr.Addr(), "", 0)

		if err := runService(db, cache); err != nil {
			t.Fatalf("runService() error = %v", err)
		}
		waitStarted(t, b.rpcStarted, "request")
		waitStarted(t, b.eventStarted, "event")
		if got := b.connects.Load(); got != 1 {
			t.Errorf("the broker was connected %d times, want 1", got)
		}
	})

	t.Run("a failure of the event queue is returned and the request consumer is already running", func(t *testing.T) {
		resetState(t)
		*config.Get() = config.Config{RabbitMQAddress: "unused"}
		b := useFakeBroker(t, errBrokerTest)
		db, _ := useSQLMock(t)
		mr := miniredis.RunT(t)
		cache := cachehandler.NewHandler(mr.Addr(), "", 0)

		err := runService(db, cache)
		if !errors.Is(err, errBrokerTest) {
			t.Errorf("runService() error = %v, want %v wrapped", err, errBrokerTest)
		}
		// listen is started first, so its consumer runs although the subscribe step failed
		waitStarted(t, b.rpcStarted, "request")
		if !notStarted(b.eventStarted) {
			t.Errorf("the event consumer was started after the event queue failed")
		}
	})
}
