package main

import (
	"net/http"
	"net/http/httptest"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-message-manager/internal/config"
)

// invalidDSN is rejected by the mysql driver while the DSN is parsed, so no connection is attempted.
const invalidDSN = "not-a-dsn"

// signalWait is an upper bound for waiting on the done channel; it is not a delay.
const signalWait = 5 * time.Second

// drainSignalChannels makes the package channels empty and stops the signal delivery registered by initSignal.
func drainSignalChannels(t *testing.T) {
	t.Helper()

	signal.Stop(chSigs)
	for {
		select {
		case <-chSigs:
			continue
		case <-chDone:
			continue
		default:
			return
		}
	}
}

// prepareSignalTest empties the channels before the test and restores that state after it.
func prepareSignalTest(t *testing.T) {
	t.Helper()

	drainSignalChannels(t)
	t.Cleanup(func() { drainSignalChannels(t) })
}

// waitDone receives from the done channel or fails the test after signalWait.
func waitDone(t *testing.T) {
	t.Helper()

	select {
	case <-chDone:
	case <-time.After(signalWait):
		t.Fatalf("Wrong match. expect: done signal, got: timeout")
	}
}

func Test_signalHandler(t *testing.T) {
	prepareSignalTest(t)

	go signalHandler()
	chSigs <- syscall.SIGTERM

	waitDone(t)
}

func Test_initSignal(t *testing.T) {
	prepareSignalTest(t)

	initSignal()
	// the signal is injected through the channel, so the test process itself is not signaled.
	chSigs <- syscall.SIGHUP

	waitDone(t)
}

func Test_initProm_disabled(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		listen   string
	}{
		// an empty endpoint must return before http.Handle, which panics for an empty pattern.
		{name: "endpoint empty", endpoint: "", listen: "203.0.113.1:2112"},
		{name: "listen empty", endpoint: "/metrics", listen: ""},
		{name: "both empty", endpoint: "", listen: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// the enabled path registers an http handler and binds a port, so it is never called here.
			initProm(tt.endpoint, tt.listen)

			// nothing may be registered on the default mux when the metrics server is disabled.
			if tt.endpoint == "" {
				return
			}
			req := httptest.NewRequest(http.MethodGet, tt.endpoint, nil)
			if _, pattern := http.DefaultServeMux.Handler(req); pattern != "" {
				t.Errorf("Wrong match. expect: no registered handler, got: %s", pattern)
			}
		})
	}
}

func Test_runService_invalidDSN(t *testing.T) {
	prepareSignalTest(t)

	origCfg := *config.Get()
	t.Cleanup(func() { *config.Get() = origCfg })
	config.Get().DatabaseDSN = invalidDSN
	config.Get().PrometheusEndpoint = ""
	config.Get().PrometheusListenAddress = ""

	err := runService()
	if err == nil || !strings.Contains(err.Error(), "invalid DSN") {
		t.Fatalf("Wrong match. expect: invalid DSN error, got: %v", err)
	}

	// runService started a signal handler through initSignal; consume it in this test.
	chSigs <- syscall.SIGTERM
	waitDone(t)
}
