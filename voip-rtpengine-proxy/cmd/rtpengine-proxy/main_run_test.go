package main

import (
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// busyListeners keeps the occupied listeners for the lifetime of the test
// binary. Closing one would let the retry goroutine of initProm bind the port
// and keep serving, so they are intentionally never closed.
var busyListeners []net.Listener

var promEndpointSeq atomic.Int64

func Test_main_unknownInterface(t *testing.T) {
	logs := captureLogs(t)

	orig := interfaceName
	interfaceName = "missing-if0"
	t.Cleanup(func() { interfaceName = orig })

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("main did not return for an unknown interface")
	}

	if logs.count(msgPrefix(logrus.ErrorLevel, "Could not get proxy ID from interface missing-if0", "")) != 1 {
		t.Errorf("Wrong match. expected one proxy ID error log")
	}
	if logs.count(msgPrefix(logrus.InfoLevel, "Proxy ID:", "")) != 0 {
		t.Errorf("Wrong match. main must return before using the proxy ID")
	}
}

func Test_initProm_listenerBusy(t *testing.T) {
	logs := captureLogs(t)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	busyListeners = append(busyListeners, l)
	addr := l.Addr().String()

	// http.Handle panics on a duplicate pattern, so every call needs its own endpoint.
	initProm(fmt.Sprintf("/test-busy-%d", promEndpointSeq.Add(1)), addr)

	logs.waitFor(t, "prometheus listener bind failure", func(e capturedEntry) bool {
		return e.level == logrus.ErrorLevel &&
			e.message == "Could not start prometheus listener" &&
			strings.Contains(fmt.Sprint(e.fields[logrus.ErrorKey]), addr)
	})
}
