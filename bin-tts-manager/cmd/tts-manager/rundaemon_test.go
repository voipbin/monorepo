package main

import (
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-tts-manager/internal/config"
)

// Test_runDaemon_databaseOpenError verifies runDaemon returns a wrapped
// error when the configured DSN cannot even be parsed by the mysql driver,
// which happens before any network connection is attempted. The test then
// drains the chSigs/chDone pair that runDaemon's initSignal call left
// behind, so the global channels and goroutine do not leak into later tests.
func Test_runDaemon_databaseOpenError(t *testing.T) {
	cfg := config.Get()
	originalDSN := cfg.DatabaseDSN
	originalPromEndpoint := cfg.PrometheusEndpoint
	originalPromListen := cfg.PrometheusListenAddress
	t.Cleanup(func() {
		cfg.DatabaseDSN = originalDSN
		cfg.PrometheusEndpoint = originalPromEndpoint
		cfg.PrometheusListenAddress = originalPromListen
	})

	cfg.DatabaseDSN = "not-a-valid-dsn"
	// Leave Prometheus disabled so initProm does not register on
	// DefaultServeMux for this test.
	cfg.PrometheusEndpoint = ""
	cfg.PrometheusListenAddress = ""

	err := runDaemon()
	if err == nil {
		t.Fatal("Wrong match. expected: error, got: nil")
	}
	if !strings.Contains(err.Error(), "could not connect to the database") {
		t.Errorf("Wrong match. expected: error to mention the database connection step, got: %v", err)
	}

	// Drain the signal goroutine that runDaemon's initSignal call started.
	chSigs <- syscall.SIGINT
	select {
	case <-chDone:
	case <-time.After(3 * time.Second):
		t.Fatal("chDone was not signalled after draining chSigs")
	}
	signal.Stop(chSigs)
}
