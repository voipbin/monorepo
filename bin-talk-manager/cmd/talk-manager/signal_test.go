package main

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

func Test_initSignal(t *testing.T) {
	tests := []struct {
		name string
		sig  syscall.Signal
	}{
		{"interrupt", syscall.SIGINT},
		{"terminate", syscall.SIGTERM},
		{"hangup", syscall.SIGHUP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// leftovers of an earlier case must not satisfy this one
			drain()

			initSignal()
			defer signal.Stop(chSigs)

			if err := syscall.Kill(os.Getpid(), tt.sig); err != nil {
				t.Fatalf("could not send the signal: %v", err)
			}

			select {
			case done := <-chDone:
				if !done {
					t.Errorf("chDone = false, want true")
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("the daemon was not told to stop after %v", tt.sig)
			}
		})
	}
}

func drain() {
	for {
		select {
		case <-chSigs:
		case <-chDone:
		default:
			return
		}
	}
}
