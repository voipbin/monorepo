package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-common-handler/models/sock"
	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-hook-manager/internal/config"
)

// fakeSock is a sock handler that outlives the test: runService may leave a listener goroutine
// behind, so a controller-based mock that ends with the test is not used here. Only Connect and
// RequestPublish are set; the other methods of the embedded interface are intentionally unset.
type fakeSock struct {
	sockhandler.SockHandler

	connects atomic.Int32
}

func (f *fakeSock) Connect() {
	f.connects.Add(1)
}

func (f *fakeSock) RequestPublish(_ context.Context, _ string, _ *sock.Request) (*sock.Response, error) {
	return &sock.Response{StatusCode: 200}, nil
}

// useFreshRegistry installs a new Prometheus registry for the call that follows, because every
// request handler creation registers collectors on the default registerer.
func useFreshRegistry(t *testing.T) {
	t.Helper()

	prev := prometheus.DefaultRegisterer
	prometheus.DefaultRegisterer = prometheus.NewRegistry()
	t.Cleanup(func() { prometheus.DefaultRegisterer = prev })
}

func Test_runService_wiring(t *testing.T) {
	const (
		testDSN    = "tcp(127.0.0.1:1)/seamtest"
		testRabbit = "amqp://rabbit-seam-test"
	)

	t.Run("database and broker get the configured addresses", func(t *testing.T) {
		prepareRunService(t)
		useFreshRegistry(t)
		cfg := config.Get()
		cfg.DatabaseDSN = testDSN
		cfg.RabbitMQAddress = testRabbit

		var gotDSN []string
		var db *sql.DB
		restoreDB := commondatabasehandler.SetConnectFunc(func(dsn string) (*sql.DB, error) {
			gotDSN = append(gotDSN, dsn)
			var err error
			db, err = sql.Open("mysql", dsn)
			return db, err
		})
		defer restoreDB()

		fake := &fakeSock{}
		var gotURIs []string
		restoreSock := sockhandler.SetRabbitFactory(func(serverURI string) sockhandler.SockHandler {
			gotURIs = append(gotURIs, serverURI)
			return fake
		})
		defer restoreSock()

		runServiceReturns(t)

		if len(gotDSN) != 1 || gotDSN[0] != testDSN {
			t.Errorf("Wrong match. expect the database to get %s once, got: %v", testDSN, gotDSN)
		}
		if len(gotURIs) != 1 || gotURIs[0] != testRabbit {
			t.Errorf("Wrong match. expect the broker to get %s once, got: %v", testRabbit, gotURIs)
		}
		if fake.connects.Load() != 1 {
			t.Errorf("Wrong match. expect: 1 broker connect, got: %d", fake.connects.Load())
		}

		// runService closes the database when it returns.
		if db == nil {
			t.Fatalf("the database was not opened")
		}
		if err := db.Ping(); err == nil || !strings.Contains(err.Error(), "closed") {
			t.Errorf("Wrong match. expect: closed database error, got: %v", err)
		}
	})

	t.Run("database failure stops before the broker", func(t *testing.T) {
		prepareRunService(t)
		useFreshRegistry(t)
		hook := captureLogs(t)

		restoreDB := commondatabasehandler.SetConnectFunc(func(string) (*sql.DB, error) {
			return nil, fmt.Errorf("connect refused for the test")
		})
		defer restoreDB()

		brokerCalls := 0
		restoreSock := sockhandler.SetRabbitFactory(func(string) sockhandler.SockHandler {
			brokerCalls++
			return &fakeSock{}
		})
		defer restoreSock()

		runServiceReturns(t)

		if brokerCalls != 0 {
			t.Errorf("Wrong match. expect: broker untouched, got: %d factory calls", brokerCalls)
		}
		if !containsMessage(funcMessages(hook, "runService"), "Could not access to database") {
			t.Errorf("Wrong match. expect: database failure log, got: %v", funcMessages(hook, "runService"))
		}
	})
}
