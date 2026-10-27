package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"monorepo/bin-customer-manager/internal/config"
)

// metricsPathCounter gives every runDaemon call its own metrics path: the path is
// registered on the process-wide http mux, which panics on a duplicate.
var metricsPathCounter atomic.Int64

// setTestConfig overrides the fields of the loaded config and restores them when the test ends.
func setTestConfig(t *testing.T, fn func(cfg *config.Config)) {
	t.Helper()

	cfg := config.Get()
	old := *cfg
	fn(cfg)
	t.Cleanup(func() {
		*config.Get() = old
	})
}

func Test_runDaemon_connectError(t *testing.T) {
	setTestConfig(t, func(cfg *config.Config) {
		// the DSN is rejected while it is parsed, so no connection is made
		cfg.DatabaseDSN = "invalid-dsn"
		cfg.PrometheusEndpoint = fmt.Sprintf("/test-metrics-%d", metricsPathCounter.Add(1))
		cfg.PrometheusListenAddr = "127.0.0.1:0"
	})
	t.Cleanup(func() {
		signal.Stop(chSigs)
	})

	err := runDaemon()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	expectPrefix := "could not connect to the database: database open error"
	if !strings.HasPrefix(err.Error(), expectPrefix) {
		t.Errorf("Wrong match. expect an error starting with: %q, got: %v", expectPrefix, err)
	}
}

func Test_initCache_connectError(t *testing.T) {
	// an address that was listening and is closed now, so the connection is refused locally
	s := miniredis.RunT(t)
	addr := s.Addr()
	s.Close()

	setTestConfig(t, func(cfg *config.Config) {
		cfg.RedisAddress = addr
	})

	res, err := initCache()
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if !strings.HasPrefix(err.Error(), "cache connect error") {
		t.Errorf("Wrong match. expect an error starting with: cache connect error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil handler, got: %v", res)
	}
}

// Test_initSignal sends the process a hangup after the handler is registered; the
// registration happens inside initSignal, so the signal never reaches the default action.
func Test_initSignal(t *testing.T) {
	t.Cleanup(func() {
		signal.Stop(chSigs)
	})

	// drop a value left by an earlier test
	select {
	case <-chDone:
	default:
	}

	initSignal()

	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("Wrong match. expect: signal sent, got: %v", err)
	}

	// the wait only bounds a failing run
	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Wrong match. expect: true, got: false")
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("Wrong match. expect: done notification, got: none")
	}
}
