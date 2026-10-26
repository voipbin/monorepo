package main

import (
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// Test_initSignal verifies that a received signal is turned into a done
// notification. No real signal is sent to the test process: the value is
// pushed to the channel the notifier writes to.
func Test_initSignal(t *testing.T) {
	// drain a stale notification, if any.
	select {
	case <-chDone:
	default:
	}

	initSignal()
	t.Cleanup(func() { signal.Stop(chSigs) })

	chSigs <- syscall.SIGTERM

	select {
	case done := <-chDone:
		if !done {
			t.Errorf("Wrong match. expect: true, got: %v", done)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("the done notification was not sent")
	}
}
