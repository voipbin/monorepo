package main

import (
	"bufio"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// testEnv holds synthetic values for every environment variable read by the
// package init(). They differ from the defaults and from each other, so a
// swapped variable mapping is detected and the result does not depend on the
// host environment.
var testEnv = map[string]string{
	"INTERFACE_NAME":            "test-if0",
	"RTPENGINE_NG_ADDRESS":      "203.0.113.10:2222",
	"RTPENGINE_NG_TIMEOUT":      "7s",
	"RABBITMQ_ADDRESS":          "amqp://test-user:test-pass@203.0.113.11:5672",
	"RABBITMQ_QUEUE_LISTEN":     "test.queue.listen",
	"REDIS_ADDRESS":             "203.0.113.12:6380",
	"REDIS_PASSWORD":            "test-redis-secret",
	"REDIS_DATABASE":            "7",
	"PROMETHEUS_ENDPOINT":       "/test-metrics",
	"PROMETHEUS_LISTEN_ADDRESS": "127.0.0.1:0",
	"RTPENGINE_RECORDING_DIR":   "/test/recording",
	"GCP_BUCKET_NAME_MEDIA":     "test-media-bucket",
}

// Package-level variables are initialized before any init() runs, so the
// environment is in place when the package init() binds its variables.
var _ = func() int {
	for k, v := range testEnv {
		if err := os.Setenv(k, v); err != nil {
			panic(err)
		}
	}
	return 0
}()

func Test_init_bindings(t *testing.T) {
	tests := []struct {
		name string

		got    string
		expect string
	}{
		{"interface name", interfaceName, testEnv["INTERFACE_NAME"]},
		{"rtpengine ng address", rtpengineNGAddress, testEnv["RTPENGINE_NG_ADDRESS"]},
		{"rtpengine ng timeout", rtpengineNGTimeout, testEnv["RTPENGINE_NG_TIMEOUT"]},
		{"rabbitmq address", rabbitMQAddress, testEnv["RABBITMQ_ADDRESS"]},
		{"rabbitmq queue listen", rabbitMQQueueListen, testEnv["RABBITMQ_QUEUE_LISTEN"]},
		{"redis address", redisAddress, testEnv["REDIS_ADDRESS"]},
		{"redis password", redisPassword, testEnv["REDIS_PASSWORD"]},
		{"redis database", strconv.Itoa(redisDatabase), testEnv["REDIS_DATABASE"]},
		{"prometheus endpoint", prometheusEndpoint, testEnv["PROMETHEUS_ENDPOINT"]},
		{"prometheus listen address", prometheusListenAddress, testEnv["PROMETHEUS_LISTEN_ADDRESS"]},
		{"recording dir", recordingDir, testEnv["RTPENGINE_RECORDING_DIR"]},
		{"gcp bucket name media", gcpBucketNameMedia, testEnv["GCP_BUCKET_NAME_MEDIA"]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, tt.got)
			}
		})
	}
}

// ifaceIPv4s returns the non-loopback IPv4 addresses of the interface.
func ifaceIPv4s(t *testing.T, iface net.Interface) []string {
	t.Helper()

	res := []string{}

	// An interface that disappeared while enumerating has no usable address.
	addrs, err := iface.Addrs()
	if err != nil {
		return res
	}

	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		res = append(res, ipNet.IP.String())
	}
	return res
}

func Test_getInterfaceIP_errors(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("could not list interfaces: %v", err)
	}

	loopbackName := ""
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 && len(ifaceIPv4s(t, iface)) == 0 {
			loopbackName = iface.Name
			break
		}
	}

	tests := []struct {
		name string

		ifaceName string
		skip      bool
	}{
		{
			name: "unknown interface",

			ifaceName: "test-no-such-if",
		},
		{
			name: "loopback only interface has no usable address",

			ifaceName: loopbackName,
			skip:      loopbackName == "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("no loopback-only interface on this host")
			}

			id, addr, err := getInterfaceIP(tt.ifaceName)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: id=%s addr=%s", id, addr)
			}
			if !strings.Contains(err.Error(), "no IPv4 address on interface") {
				t.Errorf("Wrong match. expect: no IPv4 address error, got: %v", err)
			}
			if id != "" || addr != "" {
				t.Errorf("Wrong match. expect: empty id and addr, got: %q %q", id, addr)
			}
		})
	}
}

func Test_getInterfaceIP_success(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("could not list interfaces: %v", err)
	}

	for _, iface := range ifaces {
		candidates := ifaceIPv4s(t, iface)
		if len(candidates) == 0 {
			continue
		}

		id, addr, err := getInterfaceIP(iface.Name)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if id != addr {
			t.Errorf("Wrong match. expect id equal to addr, got: %s %s", id, addr)
		}

		ip := net.ParseIP(addr)
		if ip == nil || ip.To4() == nil || ip.IsLoopback() {
			t.Fatalf("Wrong match. expect: non-loopback IPv4, got: %s", addr)
		}
		found := false
		for _, c := range candidates {
			if c == addr {
				found = true
			}
		}
		if !found {
			t.Errorf("Wrong match. expect one of %v, got: %s", candidates, addr)
		}
		return
	}

	t.Skip("no interface with a non-loopback IPv4 address on this host")
}

// fakeRedis is a loopback RESP server that answers +OK to every command and
// records the commands it received (lower-cased, space separated).
type fakeRedis struct {
	ln net.Listener

	mu       sync.Mutex
	conns    []net.Conn
	commands []string
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	f := &fakeRedis{ln: ln}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			f.mu.Lock()
			f.conns = append(f.conns, c)
			f.mu.Unlock()

			wg.Add(1)
			go func() {
				defer wg.Done()
				f.serve(c)
			}()
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		f.mu.Lock()
		for _, c := range f.conns {
			_ = c.Close()
		}
		f.mu.Unlock()
		wg.Wait()
	})
	return f
}

// serve reads RESP arrays of bulk strings and answers +OK to each one.
func (f *fakeRedis) serve(c net.Conn) {
	defer func() { _ = c.Close() }()

	r := bufio.NewReader(c)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "*") {
			return
		}
		n, err := strconv.Atoi(line[1:])
		if err != nil {
			return
		}

		args := make([]string, 0, n)
		for i := 0; i < n; i++ {
			sizeLine, err := r.ReadString('\n')
			if err != nil {
				return
			}
			sizeLine = strings.TrimSpace(sizeLine)
			if !strings.HasPrefix(sizeLine, "$") {
				return
			}
			size, err := strconv.Atoi(sizeLine[1:])
			if err != nil {
				return
			}
			buf := make([]byte, size+2)
			if _, err := io.ReadFull(r, buf); err != nil {
				return
			}
			args = append(args, string(buf[:size]))
		}

		f.mu.Lock()
		f.commands = append(f.commands, strings.ToLower(strings.Join(args, " ")))
		f.mu.Unlock()

		if _, err := c.Write([]byte("+OK\r\n")); err != nil {
			return
		}
	}
}

func (f *fakeRedis) received() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.commands...)
}

func Test_registerProxyRedis(t *testing.T) {
	f := newFakeRedis(t)

	if err := registerProxyRedis(f.ln.Addr().String(), "", 3, "10.0.0.5", "10.0.0.6"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// The server records a command before it answers, and the call above
	// waits for the answer, so the record is complete here. Only the database
	// selection and the registration are asserted on; their order and the
	// total count are not.
	for _, expect := range []string{
		"select 3",
		"set rtpengine.10.0.0.5.address-internal 10.0.0.6 ex 86400",
	} {
		found := false
		for _, cmd := range f.received() {
			if cmd == expect {
				found = true
			}
		}
		if !found {
			t.Errorf("Wrong match. expect command %q, got: %v", expect, f.received())
		}
	}
}

func Test_registerProxyRedis_unreachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("could not close listener: %v", err)
	}

	err = registerProxyRedis(addr, "", 0, "10.0.0.5", "10.0.0.6")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !strings.Contains(err.Error(), "initial Redis registration") {
		t.Errorf("Wrong match. expect: initial Redis registration error, got: %v", err)
	}
}
