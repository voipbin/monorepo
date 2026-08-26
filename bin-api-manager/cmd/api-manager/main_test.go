package main

import (
	"fmt"
	"net"
	"testing"
)

// Test_getAddressListenAudiosock verifies the AudioSocket listener always
// binds to all interfaces (an empty host in "host:port"), regardless of any
// environment configuration -- it must never be derived from the advertise
// address.
func Test_getAddressListenAudiosock(t *testing.T) {

	tests := []struct {
		name string

		expectRes string
	}{
		{
			name: "normal",

			expectRes: fmt.Sprintf(":%d", defaultAudiosockPort),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// POD_IP must have zero effect on the listen address.
			t.Setenv("POD_IP", "203.0.113.5")

			res := getAddressListenAudiosock()
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

// Test_getAddressAdvertiseAudiosock verifies the advertise address is
// resolved via nethandler.AdvertiseIP() (POD_IP override path here, since
// the auto-detect fallback depends on the host's real network interfaces
// and is already covered by nethandler's own unit tests).
func Test_getAddressAdvertiseAudiosock(t *testing.T) {

	tests := []struct {
		name string

		podIP string

		expectRes string
	}{
		{
			name: "normal ipv4",

			podIP: "203.0.113.5",

			expectRes: fmt.Sprintf("203.0.113.5:%d", defaultAudiosockPort),
		},
		{
			// Regression test: a bare fmt.Sprintf("%s:%d", ip, port) join
			// (the previous implementation) produces an unparseable string
			// for an IPv6 host -- net.JoinHostPort must be used so the
			// IPv6 host is bracketed correctly.
			name: "ipv6",

			podIP: "2001:db8::1",

			expectRes: fmt.Sprintf("[2001:db8::1]:%d", defaultAudiosockPort),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_IP", tt.podIP)

			res, err := getAddressAdvertiseAudiosock()
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}

			// Empirical proof the result is actually a dialable host:port,
			// not just a string that happens to match expectRes -- this is
			// what net.SplitHostPort/net.ResolveTCPAddr rejected before the
			// net.JoinHostPort fix for an IPv6 host.
			host, port, errSplit := net.SplitHostPort(res)
			if errSplit != nil {
				t.Fatalf("Wrong match. expect: parseable host:port, got err: %v (res: %s)", errSplit, res)
			}
			if host != tt.podIP {
				t.Errorf("Wrong match. expect host: %s, got: %s", tt.podIP, host)
			}
			if port != fmt.Sprintf("%d", defaultAudiosockPort) {
				t.Errorf("Wrong match. expect port: %d, got: %s", defaultAudiosockPort, port)
			}
			if _, errResolve := net.ResolveTCPAddr("tcp", res); errResolve != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errResolve)
			}
		})
	}
}

// Test_getAddressAdvertiseAudiosock_InvalidPodIP verifies that a POD_IP
// value which is not a well-formed IP address (e.g. a hostname, the shape
// several other services already use for their own advertise-style env
// vars) is rejected with an error instead of being silently handed to
// Asterisk. nethandler.AdvertiseIP() itself does not validate the POD_IP
// override's format (it must also serve future non-IP consumers), so this
// validation lives here, at the AudioSocket-specific consumer that requires
// a real IP for dial-back -- this is the fail-fast guard the previous
// version of this fix was missing on the POD_IP override path.
func Test_getAddressAdvertiseAudiosock_InvalidPodIP(t *testing.T) {

	tests := []struct {
		name string

		podIP string
	}{
		{
			name: "hostname instead of ip",

			podIP: "api-manager.production.svc.cluster.local",
		},
		{
			name: "garbage value",

			podIP: "not-an-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_IP", tt.podIP)

			res, err := getAddressAdvertiseAudiosock()
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil (res: %s)", res)
			}
			if res != "" {
				t.Errorf("Wrong match. expect: empty string, got: %s", res)
			}
		})
	}
}

// Test_getAddressListenAudiosock_and_Advertise_are_independent verifies the
// listen and advertise addresses never collapse into the same config value
// -- the regression this whole fix addresses (VOIP audiosocket dial-back
// bug: both used to come from the same cfg.ListenIPAudiosock field).
func Test_ListenAndAdvertiseAudiosockAddresses_areIndependent(t *testing.T) {
	t.Setenv("POD_IP", "203.0.113.5")

	listenAddress := getAddressListenAudiosock()
	advertiseAddress, err := getAddressAdvertiseAudiosock()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if listenAddress == advertiseAddress {
		t.Errorf("Wrong match. listen and advertise addresses must not be equal, got: %s", listenAddress)
	}

	expectedListen := fmt.Sprintf(":%d", defaultAudiosockPort)
	if listenAddress != expectedListen {
		t.Errorf("Wrong match. expect: %s, got: %s", expectedListen, listenAddress)
	}

	expectedAdvertise := fmt.Sprintf("203.0.113.5:%d", defaultAudiosockPort)
	if advertiseAddress != expectedAdvertise {
		t.Errorf("Wrong match. expect: %s, got: %s", expectedAdvertise, advertiseAddress)
	}
}
