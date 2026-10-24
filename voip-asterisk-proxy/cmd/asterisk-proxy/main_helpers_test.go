package main

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func Test_getPodName(t *testing.T) {
	t.Run("uses POD_NAME when set", func(t *testing.T) {
		t.Setenv("POD_NAME", "asterisk-call-test-0")

		if got := getPodName(); got != "asterisk-call-test-0" {
			t.Errorf("Wrong match. expect: asterisk-call-test-0, got: %s", got)
		}
	})

	t.Run("falls back to the hostname", func(t *testing.T) {
		t.Setenv("POD_NAME", "")

		expect, err := os.Hostname()
		if err != nil {
			t.Skipf("hostname is not available: %v", err)
		}

		if got := getPodName(); got != expect {
			t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
		}
	})
}

func Test_getPodNamespace(t *testing.T) {
	if _, err := os.Stat(inClusterNamespacePath); err == nil {
		t.Skip("the in-cluster namespace file exists on this host")
	}

	t.Run("uses POD_NAMESPACE when the namespace file is missing", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "voip-test")

		if got := getPodNamespace(); got != "voip-test" {
			t.Errorf("Wrong match. expect: voip-test, got: %s", got)
		}
	})

	t.Run("is empty when nothing is configured", func(t *testing.T) {
		t.Setenv("POD_NAMESPACE", "")

		if got := getPodNamespace(); got != "" {
			t.Errorf("Wrong match. expect: empty, got: %s", got)
		}
	})
}

func Test_setProxyInfoAnnotation_outside_cluster(t *testing.T) {
	// an empty service host makes the in-cluster config fail, so a test run
	// inside a pod never reaches a real API server.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")

	err := setProxyInfoAnnotation("11:22:33:44:55:66")
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !strings.Contains(err.Error(), "in-cluster config") {
		t.Errorf("Wrong match. expect: in-cluster config error, got: %v", err)
	}
}

func Test_getAsteriskIDAddress_errors(t *testing.T) {
	tests := []struct {
		name string

		ifaceName func(t *testing.T) string
	}{
		{
			name: "unknown interface",

			ifaceName: func(t *testing.T) string {
				return "no-such-iface0"
			},
		},
		{
			name: "loopback interface has no usable address",

			ifaceName: func(t *testing.T) string {
				ifaces, err := net.Interfaces()
				if err != nil {
					t.Skipf("interfaces are not available: %v", err)
				}
				for _, iface := range ifaces {
					if iface.Flags&net.FlagLoopback != 0 {
						return iface.Name
					}
				}
				t.Skip("no loopback interface on this host")
				return ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, addr, err := getAsteriskIDAddress(tt.ifaceName(t))
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), "no interface found") {
				t.Errorf("Wrong match. expect: no interface found, got: %v", err)
			}
			if id != "" || addr != "" {
				t.Errorf("Wrong match. expect: empty id and address, got: %q, %q", id, addr)
			}
		})
	}
}

func Test_getAsteriskIDAddress_non_loopback(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Skipf("interfaces are not available: %v", err)
	}

	// pick the first interface that has a non-loopback IPv4 address. this
	// depends on the host, so the case is skipped when there is none.
	var name, mac string
	var ips []string
	for _, iface := range ifaces {
		addrs, errAddrs := iface.Addrs()
		if errAddrs != nil {
			continue
		}

		var found []string
		for _, a := range addrs {
			ipNet, ok := a.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
				continue
			}
			found = append(found, ipNet.IP.To4().String())
		}
		if len(found) > 0 {
			name, mac, ips = iface.Name, iface.HardwareAddr.String(), found
			break
		}
	}
	if name == "" {
		t.Skip("no non-loopback IPv4 interface on this host")
	}

	gotID, gotAddr, err := getAsteriskIDAddress(name)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if gotID != mac {
		t.Errorf("Wrong id. expect the interface hardware address, got a different value")
	}
	if gotAddr != ips[0] {
		t.Errorf("Wrong address. expect the first non-loopback IPv4 address, got a different value")
	}
}

func Test_setProxyInfoRedis(t *testing.T) {
	m := miniredis.RunT(t)

	asteriskID := "11:22:33:44:55:66"
	internalAddress := "203.0.113.10"

	if err := setProxyInfoRedis(m.Addr(), "", 0, asteriskID, internalAddress); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the key is written by a background goroutine.
	key := "asterisk." + asteriskID + ".address-internal"
	deadline := time.Now().Add(5 * time.Second)
	for !m.Exists(key) {
		if time.Now().After(deadline) {
			t.Fatalf("the key was not written in time")
		}
		time.Sleep(10 * time.Millisecond)
	}

	got, err := m.Get(key)
	if err != nil {
		t.Fatalf("Could not read the key. err: %v", err)
	}
	if got != internalAddress {
		t.Errorf("Wrong value. expect: %s, got: %s", internalAddress, got)
	}
	if ttl := m.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong ttl. expect: 24h, got: %v", ttl)
	}
}

func Test_connectAMI(t *testing.T) {
	fake := newFakeAMI(t)
	host, port := fake.hostPort(t)

	username := "fake-user"
	secret := "fake-secret"
	amiSock := connectAMI(host, port, username, secret)
	t.Cleanup(func() {
		// registered after the fake server cleanup, so it runs first. the
		// client is stopped only after it has connected.
		amiSock.Close()
	})
	waitConnected(t, amiSock.Connected)

	login := fake.waitAction(t)
	if login["Action"] != "Login" || login["Username"] != username || login["Secret"] != secret {
		t.Errorf("Wrong login frame. got: Action=%s Username=%s", login["Action"], login["Username"])
	}

	res, err := amiSock.Action(map[string]string{"Action": "Ping", "ActionID": "test-action-2"})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res["Response"] != "Success" || res["ActionID"] != "test-action-2" {
		t.Errorf("Wrong response. Response=%s ActionID=%s", res["Response"], res["ActionID"])
	}
}
