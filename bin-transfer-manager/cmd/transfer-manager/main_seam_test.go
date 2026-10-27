package main

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transfer-manager/internal/config"
	"monorepo/bin-transfer-manager/pkg/cachehandler"
)

const testSeamBrokerAddress = "broker.test.invalid:5672"

// isolateMetrics gives the test its own prometheus registerer. requesthandler.NewRequestHandler
// registers its collectors on every call and panics on the second registration in a process.
func isolateMetrics(t *testing.T) {
	t.Helper()

	saved := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = saved })
}

// resetViper clears the global viper before a test and again afterwards.
func resetViper(t *testing.T) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)
}

// captureStdout runs f and returns what it wrote to os.Stdout.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}

	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = saved })

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	f()

	_ = w.Close()
	os.Stdout = saved

	return <-done
}

// seamEnv is what a test gets after seamSetup.
type seamEnv struct {
	mr       *miniredis.Miniredis
	mockSock *sockhandler.MockSockHandler
	dbs      []*sql.DB

	dsns       []string
	brokerURIs []string
}

// seamSetup installs the injection points of the common handler for the test and restores them
// when the test ends: the database connection is a lazy handle that never connects and the sock
// handler is a mock. The cache is a miniredis. The tests that use it must not run in parallel.
func seamSetup(t *testing.T) *seamEnv {
	t.Helper()

	isolateMetrics(t)

	env := &seamEnv{mr: miniredis.RunT(t)}

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)
	env.mockSock = sockhandler.NewMockSockHandler(mc)

	restoreDB := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
		env.dsns = append(env.dsns, dsn)
		db, err := sql.Open("mysql", dsn)
		env.dbs = append(env.dbs, db)
		return db, err
	})
	t.Cleanup(restoreDB)

	restoreSock := sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
		env.brokerURIs = append(env.brokerURIs, serverURI)
		return env.mockSock
	})
	t.Cleanup(restoreSock)

	// the prometheus server must stay off
	setConfig(t, config.Config{
		DatabaseDSN:             testUnreachableDSN,
		RedisAddress:            env.mr.Addr(),
		RabbitMQAddress:         testSeamBrokerAddress,
		PrometheusEndpoint:      "",
		PrometheusListenAddress: "",
	})

	// runDaemon waits for chDone: leave no value behind of an earlier test
	drainDone := func() {
		select {
		case <-chDone:
		default:
		}
	}
	drainDone()
	t.Cleanup(func() {
		signal.Stop(chSigs)
		drainDone()
	})

	return env
}

// expectConsumers expects the calls of a started subscribe handler and listen handler and returns
// the channels that are closed when each consumer goroutine has been called.
func (e *seamEnv) expectConsumers() (subscribed, listened chan struct{}) {
	subscribeQueue := string(commonoutline.QueueNameTransferSubscribe)
	exchange := string(commonoutline.QueueNameEvent)
	requestQueue := string(commonoutline.QueueNameTransferRequest)

	subscribed = make(chan struct{})
	listened = make(chan struct{})

	gomock.InOrder(
		e.mockSock.EXPECT().QueueCreate(subscribeQueue, "normal").Return(nil),
		e.mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
	)
	e.mockSock.EXPECT().QueueBind(subscribeQueue, gomock.Any(), exchange, false, nil).Return(nil).MinTimes(1)
	e.mockSock.EXPECT().ConsumeMessage(gomock.Any(), subscribeQueue, serviceName, false, false, false, 10, gomock.Any()).
		DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, numWorkers int, cb sock.CbMsgConsume) error {
			close(subscribed)
			return nil
		})

	gomock.InOrder(
		e.mockSock.EXPECT().QueueCreate(requestQueue, "normal").Return(nil),
		e.mockSock.EXPECT().ConsumeRPC(gomock.Any(), requestQueue, gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
				close(listened)
				return nil
			}),
	)

	return subscribed, listened
}

func waitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(testWaitTimeout):
		t.Fatalf("%s was not called in time", what)
	}
}

// checkClosed checks that every database handle the test opened has been closed. A handle that is
// still open fails the ping with the connection error of the unreachable DSN instead.
func (e *seamEnv) checkClosed(t *testing.T) {
	t.Helper()

	if len(e.dbs) == 0 {
		t.Fatalf("no database handle was opened")
	}
	for i, db := range e.dbs {
		err := db.PingContext(context.Background())
		if err == nil || !strings.Contains(err.Error(), "database is closed") {
			t.Errorf("handle %d: expected the handle to be closed, ping error: %v", i, err)
		}
	}
}

func newSeamCache(t *testing.T, env *seamEnv) cachehandler.CacheHandler {
	t.Helper()

	cache := cachehandler.NewHandler(env.mr.Addr(), "", 0)
	if err := cache.Connect(); err != nil {
		t.Fatalf("could not connect the test cache: %v", err)
	}
	return cache
}

func newSeamDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := sql.Open("mysql", testUnreachableDSN)
	if err != nil {
		t.Fatalf("could not open the lazy handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

func Test_run_seam(t *testing.T) {
	subscribeQueue := string(commonoutline.QueueNameTransferSubscribe)
	requestQueue := string(commonoutline.QueueNameTransferRequest)
	errSentinel := errors.New("injected failure")

	t.Run("started", func(t *testing.T) {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)
		subscribed, listened := env.expectConsumers()

		if err := run(newSeamDB(t), newSeamCache(t, env)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		waitClosed(t, subscribed, "the subscribe consumer")
		waitClosed(t, listened, "the request consumer")

		if len(env.brokerURIs) != 1 || env.brokerURIs[0] != testSeamBrokerAddress {
			t.Errorf("expected the broker address of the config to be used, got: %v", env.brokerURIs)
		}
	})

	t.Run("run twice with isolated metrics", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			env := seamSetup(t)
			env.mockSock.EXPECT().Connect().Times(1)
			subscribed, listened := env.expectConsumers()

			if err := run(newSeamDB(t), newSeamCache(t, env)); err != nil {
				t.Fatalf("call %d: unexpected error: %v", i, err)
			}
			waitClosed(t, subscribed, "the subscribe consumer")
			waitClosed(t, listened, "the request consumer")
		}
	})

	t.Run("subscribe fails", func(t *testing.T) {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)

		// no other call is expected after the failure
		env.mockSock.EXPECT().QueueCreate(subscribeQueue, "normal").Return(errSentinel)

		err := run(newSeamDB(t), newSeamCache(t, env))
		checkRunError(t, err, "could not run the subscribe handler", errSentinel)
		// the wrap of run is told from the one of runSubscribe by its own prefix
		if !strings.HasPrefix(err.Error(), "could not run the subscribe handler: could not run the subscribe handler correctly") {
			t.Errorf("expected the error of run to wrap the error of runSubscribe, got: %v", err)
		}
	})

	t.Run("listen fails", func(t *testing.T) {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)

		exchange := string(commonoutline.QueueNameEvent)
		subscribed := make(chan struct{})
		gomock.InOrder(
			env.mockSock.EXPECT().QueueCreate(subscribeQueue, "normal").Return(nil),
			env.mockSock.EXPECT().TopicCreateWithKind(exchange, "topic").Return(nil),
		)
		env.mockSock.EXPECT().QueueBind(subscribeQueue, gomock.Any(), exchange, false, nil).Return(nil).MinTimes(1)
		env.mockSock.EXPECT().ConsumeMessage(gomock.Any(), subscribeQueue, serviceName, false, false, false, 10, gomock.Any()).
			DoAndReturn(func(ctx context.Context, queueName, consumerName string, exclusive, noLocal, noWait bool, numWorkers int, cb sock.CbMsgConsume) error {
				close(subscribed)
				return nil
			})
		env.mockSock.EXPECT().QueueCreate(requestQueue, "normal").Return(errSentinel)

		err := run(newSeamDB(t), newSeamCache(t, env))
		checkRunError(t, err, "could not run the listen handler", errSentinel)
		if !strings.HasPrefix(err.Error(), "could not run the listen handler: could not run the listen handler correctly") {
			t.Errorf("expected the error of run to wrap the error of runRequestListen, got: %v", err)
		}

		// the subscribe consumer was started before the listen failure
		waitClosed(t, subscribed, "the subscribe consumer")
	})
}

func Test_runDaemon_seam(t *testing.T) {
	t.Run("stopped by the done channel", func(t *testing.T) {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)
		subscribed, listened := env.expectConsumers()

		// the stop request is already there when the daemon waits for it
		chDone <- true

		if err := runDaemon(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		waitClosed(t, subscribed, "the subscribe consumer")
		waitClosed(t, listened, "the request consumer")

		if len(env.dsns) != 1 || env.dsns[0] != testUnreachableDSN {
			t.Errorf("expected the database to be opened with the DSN of the config, got: %v", env.dsns)
		}
		select {
		case <-chDone:
			t.Errorf("the done channel must be empty after the stop")
		default:
		}
		env.checkClosed(t)
	})

	t.Run("cache failure after the database", func(t *testing.T) {
		env := seamSetup(t)

		addr := env.mr.Addr()
		env.mr.Close()
		config.Get().RedisAddress = addr

		err := runDaemon()
		if err == nil {
			t.Fatalf("expected an error, got nil")
		}
		if !strings.HasPrefix(err.Error(), "could not initialize the cache") || !strings.Contains(err.Error(), "cache connect error") {
			t.Errorf("unexpected error: %v", err)
		}
		if len(env.dsns) != 1 {
			t.Errorf("expected the database to be opened once before the cache failure, got: %v", env.dsns)
		}
		if len(env.brokerURIs) != 0 {
			t.Errorf("the broker must not be used after the cache failure, got: %v", env.brokerURIs)
		}
		env.checkClosed(t)
	})

	t.Run("run fails", func(t *testing.T) {
		env := seamSetup(t)
		env.mockSock.EXPECT().Connect().Times(1)

		errSentinel := errors.New("injected failure")
		env.mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameTransferSubscribe), "normal").Return(errSentinel)

		err := runDaemon()
		checkRunError(t, err, "could not run transfer-manager", errSentinel)
		env.checkClosed(t)
	})
}

func Test_main_seam(t *testing.T) {
	savedArgs := os.Args
	t.Cleanup(func() { os.Args = savedArgs })
	os.Args = []string{"transfer-manager", "completion", "bash"}

	savedFormatter := logrus.StandardLogger().Formatter
	savedLevel := logrus.GetLevel()
	t.Cleanup(func() {
		logrus.SetFormatter(savedFormatter)
		logrus.SetLevel(savedLevel)
	})
	resetViper(t)

	// the pre-run hook loads the config once per process: keep the values the same either way
	saved := *config.Get()
	t.Cleanup(func() { *config.Get() = saved })

	out := captureStdout(t, main)
	if !strings.Contains(out, "bash completion") {
		t.Errorf("expected the completion script of the command in the output, got %d bytes", len(out))
	}
}
