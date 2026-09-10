package mcpserverhandler

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// McpHTTPResponseSizeCapBytes bounds the response body an SSRF-guarded MCP
// client will read, so even an allowed host cannot be used to exhaust a pod
// (design §7 "Enforce a response size cap ... regardless of the above
// checks"). Callers should wrap the response body in io.LimitReader(body,
// McpHTTPResponseSizeCapBytes) before decoding.
const McpHTTPResponseSizeCapBytes = 1 << 20 // 1 MiB

// ValidateURL enforces the static (create/update-time and pre-dial)
// checks from design §7: https-only scheme, and -- for a literal IP host --
// rejection of private/loopback/link-local/multicast ranges via net.IP's
// built-in classifiers (NOT a hand-rolled CIDR list, which would silently
// miss an IPv4-mapped IPv6 form such as ::ffff:169.254.169.254).
//
// This is necessary but not sufficient on its own: a hostname (as opposed to
// a literal IP) can resolve to a public address here and a private one at
// dial time (DNS rebinding). NewSSRFGuardedClient's dial-time Control hook
// closes that gap; call both, not just this one, at every outbound call site
// (design §7 Round 2/3 findings).
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("url scheme must be https, got %q", u.Scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url has no host")
	}

	// A literal IP host can be checked immediately; a hostname is deferred to
	// dial time (this call site cannot resolve DNS safely/synchronously here
	// without duplicating the dial-time re-check, and design §7 explicitly
	// requires the dial-time check regardless).
	if ip := net.ParseIP(host); ip != nil {
		if err := rejectDisallowedIP(ip); err != nil {
			return err
		}
	}

	return nil
}

// rejectDisallowedIP returns an error if ip is private, loopback,
// link-local (unicast or multicast), or unspecified. Uses net.IP's built-in
// classifiers, which correctly unwrap IPv4-mapped IPv6 addresses via To4()
// internally (design §7 Round 2 finding, MAJOR).
func rejectDisallowedIP(ip net.IP) error {
	switch {
	case ip.IsPrivate():
		return fmt.Errorf("ip %s is a private address", ip)
	case ip.IsLoopback():
		return fmt.Errorf("ip %s is a loopback address", ip)
	case ip.IsLinkLocalUnicast():
		return fmt.Errorf("ip %s is a link-local unicast address (includes the cloud metadata address)", ip)
	case ip.IsLinkLocalMulticast():
		return fmt.Errorf("ip %s is a link-local multicast address", ip)
	case ip.IsUnspecified():
		return fmt.Errorf("ip %s is the unspecified address", ip)
	}
	return nil
}

// NewSSRFGuardedClient builds an *http.Client whose Dialer.Control hook
// validates the ACTUAL socket peer address the dialer is about to connect
// to, rejecting the dial itself on failure -- not a detached pre-flight
// LookupHost whose result the transport is free to ignore.
//
// This is the dial-time pinning requirement from design §7 Round 3/4: a
// multi-A-record host could pass a separate pre-flight check on one IP while
// the client dials a different (unvalidated) one, and a TTL=0 DNS record can
// flip between lookup and dial even within one "call time" window.
// net.Dialer.Control is the correct primitive because it runs as a callback
// during the dial http.Transport already performs, inspecting the resolved
// peer address WITHOUT altering what address is dialed or what hostname is
// used for the TLS ServerName (SNI) -- so certificate validation against the
// original hostname is unaffected. Every resolved IP for a hostname is
// subject to this check: http.Transport retries each address in the
// resolver's list in turn, and Control runs again for each attempt.
func NewSSRFGuardedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: controlRejectDisallowedAddr,
	}

	transport := &http.Transport{
		DialContext: dialer.DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}

// controlRejectDisallowedAddr is the net.Dialer.Control hook: address is the
// actual resolved peer address the dialer is about to connect to. Returning
// a non-nil error aborts that specific dial attempt.
func controlRejectDisallowedAddr(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("could not parse dial address %q: %w", address, err)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("dial address %q did not resolve to a literal IP", address)
	}

	if err := rejectDisallowedIP(ip); err != nil {
		return fmt.Errorf("rejected dial to %s (%s): %w", address, network, err)
	}

	return nil
}
