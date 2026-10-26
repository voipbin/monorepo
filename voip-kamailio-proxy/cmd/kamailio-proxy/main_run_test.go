package main

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

var brokerQueueSeq atomic.Int64

// firstMACInterface returns the first interface that has a hardware address,
// the same condition getKamailioID needs to derive the instance identity.
func firstMACInterface() (name, mac string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", ""
	}
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) > 0 {
			return iface.Name, iface.HardwareAddr.String()
		}
	}
	return "", ""
}

func drainSigs() {
	for {
		select {
		case <-chSigs:
		default:
			return
		}
	}
}

// Test_main_declaresAndConsumesQueues runs main against an in-process broker
// and verifies the queues it declares, the bindings and consumers it starts,
// and that it returns once a signal arrives.
func Test_main_declaresAndConsumesQueues(t *testing.T) {
	ifaceName, mac := firstMACInterface()
	if ifaceName == "" {
		t.Skip("no network interface with a hardware address")
	}

	hook := isolateLogger(t)
	broker := startFakeBroker(t)

	drainSigs()
	t.Cleanup(drainSigs)

	permanent := fmt.Sprintf("test.kamailio-proxy.%d.request", brokerQueueSeq.Add(1))
	volatile := fmt.Sprintf("voip.kamailio.%s.request", mac)

	savedAddress, savedQueue, savedInterface := rabbitMQAddress, rabbitMQQueueListen, interfaceName
	rabbitMQAddress, rabbitMQQueueListen, interfaceName = broker.uri(), permanent, ifaceName

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	// stop ends main through the signal channel and waits for it to return.
	stop := func() bool {
		select {
		case chSigs <- os.Interrupt:
		default:
		}
		select {
		case <-done:
			return true
		case <-time.After(5 * time.Second):
			return false
		}
	}

	// Wait for both consumers; the broker state, not elapsed time, decides.
	var consumers []startedConsumer
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_, _, _, consumers, _ = broker.snapshot()
		if len(consumers) >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if len(consumers) < 2 {
		if !stop() {
			// main is still running, so leave the globals alone to avoid racing it.
			t.Fatalf("main did not return after the consumers failed to start")
		}
		rabbitMQAddress, rabbitMQQueueListen, interfaceName = savedAddress, savedQueue, savedInterface
		t.Fatalf("Expected 2 consumers, got %d", len(consumers))
	}

	queues, exchanges, binds, consumers, problems := broker.snapshot()

	if !stop() {
		t.Fatalf("main did not return after a signal")
	}
	rabbitMQAddress, rabbitMQQueueListen, interfaceName = savedAddress, savedQueue, savedInterface

	for _, p := range problems {
		t.Errorf("Fake broker problem: %s", p)
	}

	// permanent queue: durable only, no expiry.
	// volatile queue: auto-delete only, expires after 30 minutes without consumers.
	// Messages name the queue by role because the volatile name embeds the host MAC.
	roles := []struct {
		role  string
		name  string
		flags byte
	}{
		{"permanent", permanent, queueFlagDurable},
		{"volatile", volatile, queueFlagAutoDelete},
	}
	for _, r := range roles {
		var found bool
		for _, q := range queues {
			if q.name != r.name {
				continue
			}
			found = true
			if q.flags != r.flags {
				t.Errorf("The %s queue was declared with flags 0x%02x, want 0x%02x", r.role, q.flags, r.flags)
			}
			expires, hasExpires := q.args["x-expires"]
			if r.role == "permanent" && hasExpires {
				t.Errorf("The permanent queue must not expire, got x-expires %v", expires)
			}
			if r.role == "volatile" && expires != int32(1800000) {
				t.Errorf("The volatile queue x-expires = %v, want int32 1800000", expires)
			}
		}
		if !found {
			t.Errorf("The %s queue was never declared", r.role)
		}
	}

	var delayExchange bool
	for _, e := range exchanges {
		if e.name == "bin-manager.delay" && e.kind == "x-delayed-message" && e.flags == exchangeFlagDurable && e.args["x-delayed-type"] == "direct" {
			delayExchange = true
		}
	}
	if !delayExchange {
		t.Errorf("Expected a durable x-delayed-message exchange bin-manager.delay, got %v", exchanges)
	}

	for _, r := range roles {
		var bound, consumed bool
		for _, b := range binds {
			if b.queue == r.name && b.key == r.name && b.exchange == "bin-manager.delay" {
				bound = true
			}
		}
		for _, c := range consumers {
			if c.queue != r.name {
				continue
			}
			consumed = true
			if c.tag != "kamailio-proxy" {
				t.Errorf("The %s queue consumer tag = %q, want kamailio-proxy", r.role, c.tag)
			}
			if c.prefetch != 10 {
				t.Errorf("The %s queue prefetch = %d, want 10", r.role, c.prefetch)
			}
		}
		if !bound {
			t.Errorf("The %s queue was not bound to the delay exchange with its own name as key", r.role)
		}
		if !consumed {
			t.Errorf("The %s queue has no consumer", r.role)
		}
	}

	waitLogEntry(t, hook, "Terminating kamailio-proxy", "interrupt")
}
