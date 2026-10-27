package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"monorepo/bin-conversation-manager/internal/config"
)

// unreachableDSN points to a local port that refuses the connection at once, so no test depends on a real database.
const unreachableDSN = "tcp(127.0.0.1:1)/db"

func Test_runMain_databaseFailure(t *testing.T) {
	// the global configuration is loaded once per process, so the values are set on it directly
	// and restored, which does not depend on the order of the tests
	seamGlobals(t)
	config.Get().DatabaseDSN = unreachableDSN
	seamRegistry(t)

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
