// Package nethandler provides helpers for resolving network addresses that
// a process should advertise to peers, as opposed to the address it binds
// its listening socket to.
//
// The distinction matters for bin-api-manager's AudioSocket/ExternalMedia
// listener, which binds a socket to all interfaces (0.0.0.0:<port>) but
// must hand a dialable host:port to a remote peer (Asterisk) so it can
// connect back. Binding and advertising must never share a single
// "listen IP" config value: an advertise address needs a real, routable
// host, while a listen address is free to be the wildcard.
//
// This package is internal to bin-api-manager (see the bin-common-handler
// admission rule in the root CLAUDE.md: a package may only live in
// bin-common-handler if it is used by 3+ services). Should
// bin-pipecat-manager, bin-tts-manager, or bin-transcribe-manager later need
// the same POD_IP-for-advertise-address resolution, promoting this package
// to bin-common-handler at that time is the correct procedure.
package nethandler

import (
	"net"
	"os"

	"github.com/pkg/errors"
)

// EnvPodIP is the environment variable that, when set, overrides the
// auto-detected advertise IP. This is how Kubernetes' Downward API
// (fieldRef: status.podIP) and any Docker Compose deployment that sets it
// explicitly communicate the pod/container's externally-reachable address.
//
// Exported so callers that need to know (e.g. for a startup log line
// describing which resolution path was used) can reference the same
// constant AdvertiseIP() consults, instead of re-declaring the literal.
const EnvPodIP = "POD_IP"

// Test seams. Production code always uses the real os/net implementations;
// tests override these package vars to make interface enumeration and
// environment lookups deterministic.
var (
	osGetenv         = os.Getenv
	netInterfacesFn  = net.Interfaces
	interfaceAddrsFn = func(iface net.Interface) ([]net.Addr, error) { return iface.Addrs() }
)

// AdvertiseIP returns the IP address other hosts (e.g. Asterisk dialing back
// for AudioSocket/ExternalMedia streaming) should use to reach this process.
//
// This is deliberately distinct from a listen address: a listen address may
// be the wildcard (0.0.0.0) so the process accepts connections on every
// interface, but an advertise address must be a concrete, routable host.
//
// Resolution order:
//  1. POD_IP environment variable, if set -- returned verbatim. This is the
//     explicit override path (Kubernetes Downward API, or an operator-set
//     Docker Compose environment variable).
//  2. Auto-detected non-loopback IPv4 address from the host's network
//     interfaces (e.g. the container's `production` Docker network
//     interface in Komodo/Compose deployments that do not set POD_IP).
//
// If neither source yields an address, AdvertiseIP returns an error so
// callers fail fast instead of silently advertising an empty/unusable host.
func AdvertiseIP() (string, error) {
	if podIP := osGetenv(EnvPodIP); podIP != "" {
		return podIP, nil
	}

	ip, err := autoDetectIP()
	if err != nil {
		return "", errors.Wrap(err, "could not auto-detect advertise ip")
	}

	return ip, nil
}

// autoDetectIP scans the host's network interfaces for the first
// non-loopback, non-link-local IPv4 address.
func autoDetectIP() (string, error) {
	ifaces, err := netInterfacesFn()
	if err != nil {
		return "", errors.Wrap(err, "could not list network interfaces")
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, errAddrs := interfaceAddrsFn(iface)
		if errAddrs != nil {
			continue
		}

		if ip := firstUsableIPv4(addrs); ip != "" {
			return ip, nil
		}
	}

	return "", errors.New("no non-loopback ipv4 address found on any network interface")
}

// firstUsableIPv4 returns the first non-loopback, non-link-local IPv4
// address string found in addrs, or "" if none qualify.
func firstUsableIPv4(addrs []net.Addr) string {
	for _, addr := range addrs {
		ip := extractIP(addr)
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}

		ip4 := ip.To4()
		if ip4 == nil {
			// IPv6-only address -- not usable here.
			continue
		}

		return ip4.String()
	}

	return ""
}

// extractIP pulls the net.IP out of the concrete net.Addr implementations
// returned by net.Interface.Addrs (*net.IPNet in practice, *net.IPAddr for
// completeness).
func extractIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}
