package ai

import (
	"net/netip"
)

// The block table of customer specified endpoints (design appendix A). The Python runner
// keeps an identical table in custom_endpoint_net.py; both are checked against
// testdata/endpoint_blocklist.json.

func mustPrefixes(in ...string) []netip.Prefix {
	res := make([]netip.Prefix, 0, len(in))
	for _, s := range in {
		res = append(res, netip.MustParsePrefix(s))
	}

	return res
}

var blockedV4Prefixes = mustPrefixes(
	"0.0.0.0/8",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"100.64.0.0/10",
	"198.18.0.0/15",
	"192.0.0.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"192.0.2.0/24",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"192.88.99.0/24",
	"192.175.48.0/24",
)

var blockedV6Prefixes = mustPrefixes(
	"::/128",
	"::1/128",
	"fc00::/7",
	"fe80::/10",
	"ff00::/8",
	"64:ff9b:1::/48",
	"100::/64",
	"::/96",
	"::ffff:0:0:0/96",
	"fec0::/10",
	"2001::/32",
	"2001:db8::/32",
	"2001:2::/48",
	"3fff::/20",
)

// nat64Prefix and sixToFourPrefix are not refused as such: the embedded IPv4 decides.
var (
	nat64Prefix     = netip.MustParsePrefix("64:ff9b::/96")
	sixToFourPrefix = netip.MustParsePrefix("2002::/16")
)

func inPrefixes(a netip.Addr, list []netip.Prefix) bool {
	for _, p := range list {
		if p.Contains(a) {
			return true
		}
	}

	return false
}

// isBlockedAddr reports whether a connection to the address must be refused.
func isBlockedAddr(a netip.Addr) bool {
	if !a.IsValid() {
		return true
	}
	a = a.WithZone("")

	if a.Is4In6() {
		return inPrefixes(a.Unmap(), blockedV4Prefixes)
	}
	if a.Is4() {
		return inPrefixes(a, blockedV4Prefixes)
	}

	b := a.As16()
	switch {
	case nat64Prefix.Contains(a):
		return inPrefixes(netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]}), blockedV4Prefixes)
	case sixToFourPrefix.Contains(a):
		return inPrefixes(netip.AddrFrom4([4]byte{b[2], b[3], b[4], b[5]}), blockedV4Prefixes)
	}

	return inPrefixes(a, blockedV6Prefixes)
}

// isBlockedAddrString parses and checks an address; anything unparseable is blocked (fail closed).
func isBlockedAddrString(s string) bool {
	a, err := netip.ParseAddr(s)
	if err != nil {
		return true
	}

	return isBlockedAddr(a)
}
