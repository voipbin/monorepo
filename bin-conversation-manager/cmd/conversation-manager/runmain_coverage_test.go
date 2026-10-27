package main

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"monorepo/bin-conversation-manager/internal/config"
)

// unreachableDSN points to a local port that refuses the connection at once, so no test depends on a real database.
const unreachableDSN = "tcp(127.0.0.1:1)/db"

func Test_runMain_databaseFailure(t *testing.T) {
	// the configuration is loaded once per process from the environment, so the values are injected before the first call
	t.Setenv("DATABASE_DSN", unreachableDSN)
	t.Setenv("PROMETHEUS_LISTEN_ADDRESS", "127.0.0.1:0")

	// the global configuration is locked after the first load. Any other value means the environment
	// was not applied and the run could reach a real database.
	if dsn := config.Get().DatabaseDSN; dsn != "" && dsn != unreachableDSN {
		t.Fatalf("the global configuration is already loaded with another database address")
	}

	// initProm registers the metrics handler on the default mux, which panics on a second registration
	prevMux := http.DefaultServeMux
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() {
		http.DefaultServeMux = prevMux
		signal.Reset(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	})

	err := runMain(rootCmd, nil)
	if err == nil {
		t.Fatalf("runMain() error = nil, want the database connection error")
	}
	if !strings.Contains(err.Error(), "127.0.0.1:1") {
		t.Errorf("runMain() error = %q, want the refused address of the injected database", err.Error())
	}
}

func Test_signalHandler(t *testing.T) {
	// a handler left by an earlier test may receive the signal instead; either way one done value is sent
	select {
	case <-chDone:
	default:
	}

	go signalHandler()
	chSigs <- os.Interrupt

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("chDone = false, want true")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the done channel did not receive a value after the signal")
	}
}
