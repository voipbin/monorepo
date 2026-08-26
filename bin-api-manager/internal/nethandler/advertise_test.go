package nethandler

import (
	"errors"
	"net"
	"testing"
)

var errNetInterfaces = errors.New("boom: could not enumerate interfaces")

// resetSeams restores the package-level test seams to values that will not
// leak between test cases.
func resetSeams(t *testing.T) {
	t.Helper()

	origGetenv := osGetenv
	origInterfaces := netInterfacesFn
	origAddrs := interfaceAddrsFn

	t.Cleanup(func() {
		osGetenv = origGetenv
		netInterfacesFn = origInterfaces
		interfaceAddrsFn = origAddrs
	})
}

func Test_AdvertiseIP_PodIPOverride(t *testing.T) {
	resetSeams(t)

	osGetenv = func(key string) string {
		if key == EnvPodIP {
			return "203.0.113.10"
		}
		return ""
	}

	// Auto-detect must never be consulted when POD_IP is set -- force it to
	// fail loudly if it is.
	netInterfacesFn = func() ([]net.Interface, error) {
		t.Fatal("netInterfacesFn should not be called when POD_IP is set")
		return nil, nil
	}

	res, err := AdvertiseIP()
	if err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}

	if res != "203.0.113.10" {
		t.Errorf("Wrong match. expected: 203.0.113.10, got: %s", res)
	}
}

func Test_AdvertiseIP_AutoDetectSuccess(t *testing.T) {
	resetSeams(t)

	osGetenv = func(key string) string { return "" }

	netInterfacesFn = func() ([]net.Interface, error) {
		return []net.Interface{
			{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
			{Name: "eth0", Flags: net.FlagUp},
		}, nil
	}

	interfaceAddrsFn = func(iface net.Interface) ([]net.Addr, error) {
		switch iface.Name {
		case "lo":
			return []net.Addr{&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}}, nil
		case "eth0":
			return []net.Addr{
				// link-local address should be skipped before the real one
				&net.IPNet{IP: net.ParseIP("169.254.1.5"), Mask: net.CIDRMask(16, 32)},
				&net.IPNet{IP: net.ParseIP("172.20.0.5"), Mask: net.CIDRMask(24, 32)},
			}, nil
		default:
			return nil, nil
		}
	}

	res, err := AdvertiseIP()
	if err != nil {
		t.Fatalf("Wrong match. expected: nil, got: %v", err)
	}

	if res != "172.20.0.5" {
		t.Errorf("Wrong match. expected: 172.20.0.5, got: %s", res)
	}
}

func Test_AdvertiseIP_FailFast(t *testing.T) {

	type test struct {
		name             string
		netInterfacesFn  func() ([]net.Interface, error)
		interfaceAddrsFn func(net.Interface) ([]net.Addr, error)
	}

	tests := []test{
		{
			name: "no interfaces at all",
			netInterfacesFn: func() ([]net.Interface, error) {
				return []net.Interface{}, nil
			},
			interfaceAddrsFn: func(iface net.Interface) ([]net.Addr, error) {
				return nil, nil
			},
		},
		{
			name: "only loopback and link-local addresses",
			netInterfacesFn: func() ([]net.Interface, error) {
				return []net.Interface{
					{Name: "lo", Flags: net.FlagUp | net.FlagLoopback},
					{Name: "eth0", Flags: net.FlagUp},
				}, nil
			},
			interfaceAddrsFn: func(iface net.Interface) ([]net.Addr, error) {
				switch iface.Name {
				case "lo":
					return []net.Addr{&net.IPNet{IP: net.ParseIP("127.0.0.1"), Mask: net.CIDRMask(8, 32)}}, nil
				case "eth0":
					return []net.Addr{&net.IPNet{IP: net.ParseIP("169.254.3.9"), Mask: net.CIDRMask(16, 32)}}, nil
				default:
					return nil, nil
				}
			},
		},
		{
			name: "interface enumeration itself errors",
			netInterfacesFn: func() ([]net.Interface, error) {
				return nil, errNetInterfaces
			},
			interfaceAddrsFn: func(iface net.Interface) ([]net.Addr, error) {
				return nil, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetSeams(t)

			osGetenv = func(key string) string { return "" }
			netInterfacesFn = tt.netInterfacesFn
			interfaceAddrsFn = tt.interfaceAddrsFn

			res, err := AdvertiseIP()
			if err == nil {
				t.Fatalf("Wrong match. expected: error, got: nil (res: %s)", res)
			}

			if res != "" {
				t.Errorf("Wrong match. expected: empty string, got: %s", res)
			}
		})
	}
}
