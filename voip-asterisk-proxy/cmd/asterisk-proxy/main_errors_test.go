package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func Test_connectAMI_logsConnectionError(t *testing.T) {
	fake := newFakeAMI(t)
	host, port := fake.hostPort(t)

	amiSock := connectAMI(host, port, "fake-user", "fake-secret")
	t.Cleanup(func() {
		// registered after the fake server cleanup, so it runs first.
		amiSock.Close()
	})
	waitConnected(t, amiSock.Connected)

	logs := captureLogs(t)
	fake.dropConns()

	// the client reports the lost connection asynchronously.
	logs.waitFor(t, "AMI error log for the dropped connection", func(e capturedEntry) bool {
		return strings.HasPrefix(e.message, "AMI error: ") && strings.Contains(e.message, "AMI TCP ERROR")
	})

	// wait for the reconnect before Close so Close does not race with it.
	waitConnected(t, amiSock.Connected)
}

func Test_setProxyInfoRedis_logsSetFailure(t *testing.T) {
	// an address that was just released, so the connection is refused.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	// unique per run: the update goroutine cannot be stopped, so an older
	// run must not be able to satisfy this run's wait.
	asteriskID := fmt.Sprintf("fail-%d", time.Now().UnixNano())
	internalAddress := "203.0.113.20"

	logs := captureLogs(t)
	if errSet := setProxyInfoRedis(addr, "", 0, asteriskID, internalAddress); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}

	e := logs.waitFor(t, "set failure log for this call", func(e capturedEntry) bool {
		return strings.HasPrefix(e.message, "Could not set internal address") &&
			e.data["asterisk_id"] == asteriskID &&
			e.data["internal_address"] == internalAddress
	})
	if !strings.Contains(e.message, "err:") {
		t.Errorf("Wrong message. expect: the redis error to be included, got: %s", e.message)
	}
}
