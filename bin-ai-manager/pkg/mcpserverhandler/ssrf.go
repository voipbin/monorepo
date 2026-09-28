package mcpserverhandler

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// McpHTTPResponseSizeCapBytes bounds the response body an SSRF-guarded MCP
// client will read, so even an allowed host cannot be used to exhaust a pod
// (design §7 "Enforce a response size cap ... regardless of the above
// checks"). Callers should wrap the response body in io.LimitReader(body,
// McpHTTPResponseSizeCapBytes) before decoding.
const McpHTTPResponseSizeCapBytes = 1 << 20 // 1 MiB

// mcpMaxResponseHeaderBytes bounds the response headers an SSRF-guarded MCP
// client accepts from a customer server, in place of net/http's 10 MB default.
const mcpMaxResponseHeaderBytes = 64 << 10 // 64 KiB

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
//
// The client also refuses redirects (RefuseRedirects) and any request that is
// not https (httpsOnlyTransport), so a credential is only ever sent, over TLS,
// to the host the customer registered.
func NewSSRFGuardedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: controlRejectDisallowedAddr,
	}
	return newGuardedClient(timeout, dialer.DialContext, &tls.Config{MinVersion: tls.VersionTLS12})
}

// newGuardedClient assembles the client around a given dialer and TLS
// configuration, so tests can reach a local TLS server through everything
// else the production client does.
func newGuardedClient(timeout time.Duration, dial func(ctx context.Context, network, addr string) (net.Conn, error), tlsConfig *tls.Config) *http.Client {
	transport := &http.Transport{
		DialContext:     dial,
		TLSClientConfig: tlsConfig,
		// The peer is a customer-controlled server. Bound its response
		// headers well below net/http's 10 MB default: the MCP client echoes
		// one of them (Mcp-Session-Id) back on every later request.
		MaxResponseHeaderBytes: mcpMaxResponseHeaderBytes,
	}

	return &http.Client{
		Transport:     &httpsOnlyTransport{next: transport},
		Timeout:       timeout,
		CheckRedirect: RefuseRedirects,
	}
}

// ErrRedirectRefused is returned, wrapped, when an MCP server answers with a
// redirect.
var ErrRedirectRefused = errors.New("mcp server answered with a redirect, which is refused")

// RefuseRedirects is an http.Client CheckRedirect policy that follows no
// redirect.
//
// Following one hands net/http the decision of which headers go to the new
// location, and that decision is wrong for this client in two proven ways:
// a custom API-key header is not on net/http's sensitive list and reaches a
// different host intact, and Authorization survives an https to http
// downgrade on the same host because net/http compares host names only. Re-
// checking each hop would mean re-implementing that decision. An MCP endpoint
// is a URL the customer registered, so a server redirecting it is
// misconfigured, and refusing gives a clear error instead of a silent leak.
//
// The error names the target by scheme and host only, since the path and
// query of a URL the server chose may carry anything. net/http wraps it in a
// *url.Error carrying the full target URL, so callers must report it through
// RedirectRefusal rather than the wrapped error's text. The one common cause
// gets a specific hint: the reference MCP SDK answers the registered path with
// a trailing slash added or removed by redirecting to the other spelling.
func RefuseRedirects(req *http.Request, via []*http.Request) error {
	target := req.URL
	hint := ""
	if len(via) > 0 {
		orig := via[0].URL
		if target.Scheme == orig.Scheme && target.Host == orig.Host && target.Path != orig.Path &&
			strings.TrimSuffix(target.Path, "/") == strings.TrimSuffix(orig.Path, "/") {
			hint = "; the server serves the registered path with a different trailing slash, so update the registered URL to match"
		}
	}
	return fmt.Errorf("%w: to %s://%s%s", ErrRedirectRefused, target.Scheme, target.Host, hint)
}

// RedirectRefusal returns the refusal from RefuseRedirects inside err, without
// the *url.Error net/http wraps it in, whose text includes the full redirect
// target. ok is false when err is not a refused redirect.
func RedirectRefusal(err error) (refusal error, ok bool) {
	if !errors.Is(err, ErrRedirectRefused) {
		return nil, false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err, true
	}
	return err, true
}

// httpsOnlyTransport refuses any request whose scheme is not https before it
// is sent. URLs are validated as https when they are stored, but not again
// before each call, so without this a row holding an http URL would send its
// credential in cleartext on the first request.
type httpsOnlyTransport struct {
	next *http.Transport
}

func (t *httpsOnlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, fmt.Errorf("refusing a %q request: mcp servers are reached over https only", req.URL.Scheme)
	}
	return t.next.RoundTrip(req)
}

// CloseIdleConnections forwards to the wrapped transport. http.Client calls
// it through an interface check, so without it a caller closing idle
// connections would silently close nothing.
func (t *httpsOnlyTransport) CloseIdleConnections() {
	t.next.CloseIdleConnections()
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
