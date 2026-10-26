package main

import (
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// Test_initSignal_deliversSignalToChDone verifies initSignal's goroutine
// forwards a single injected signal onto chDone, and that signal.Stop undoes
// the registration afterward so the global channels do not leak across
// tests. No real OS signal is sent.
func Test_initSignal_deliversSignalToChDone(t *testing.T) {
	initSignal()
	t.Cleanup(func() {
		signal.Stop(chSigs)
	})

	chSigs <- syscall.SIGINT

	select {
	case <-chDone:
	case <-time.After(3 * time.Second):
		t.Fatal("chDone was not signalled")
	}
}
