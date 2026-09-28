package mcpserverhandler

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func Test_ValidateURL_RejectsPlainHTTP(t *testing.T) {
	if err := ValidateURL("http://example.com/mcp"); err == nil {
		t.Errorf("expected error for plain http scheme, got nil")
	}
}

func Test_ValidateURL_RejectsOtherSchemes(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "gopher://example.com", "ftp://example.com"} {
		if err := ValidateURL(u); err == nil {
			t.Errorf("expected error for scheme in %q, got nil", u)
		}
	}
}

func Test_ValidateURL_AcceptsHTTPSHostname(t *testing.T) {
	if err := ValidateURL("https://example.com/mcp"); err != nil {
		t.Errorf("expected no error for a well-formed https hostname url, got %v", err)
	}
}

func Test_ValidateURL_RejectsLiteralPrivateIP(t *testing.T) {
	tests := []string{
		"https://127.0.0.1/mcp",
		"https://169.254.169.254/mcp", // cloud metadata address
		"https://10.0.0.1/mcp",
		"https://[::1]/mcp",
	}
	for _, u := range tests {
		if err := ValidateURL(u); err == nil {
			t.Errorf("expected error for private/loopback/link-local url %q, got nil", u)
		}
	}
}

func Test_rejectDisallowedIP_TableTest(t *testing.T) {
	tests := []struct {
		name      string
		ip        string
		wantError bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"private class A", "10.0.0.1", true},
		{"private class C", "192.168.1.1", true},
		{"link-local (cloud metadata)", "169.254.169.254", true},
		{"link-local multicast", "224.0.0.1", true},
		{"unspecified v4", "0.0.0.0", true},
		// IPv4-mapped IPv6 form of the metadata address -- must be rejected
		// via net.IP's built-in classifiers, which correctly unwrap this via
		// To4() internally (design §7 Round 2 finding).
		{"IPv4-mapped IPv6 metadata address", "::ffff:169.254.169.254", true},
		{"IPv4-mapped IPv6 loopback", "::ffff:127.0.0.1", true},
		{"IPv4-mapped IPv6 private", "::ffff:10.0.0.1", true},
		// Public addresses must be accepted.
		{"public v4 (google dns)", "8.8.8.8", false},
		{"public v4 (cloudflare dns)", "1.1.1.1", false},
		{"public v6", "2606:4700:4700::1111", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("test setup error: could not parse ip %q", tt.ip)
			}
			err := rejectDisallowedIP(ip)
			if tt.wantError && err == nil {
				t.Errorf("expected an error rejecting ip %q, got nil", tt.ip)
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error for public ip %q, got %v", tt.ip, err)
			}
		})
	}
}

// Test_NewSSRFGuardedClient_RejectsDialToPrivateIP exercises the actual
// dial-time Control hook (design §7 Round 3/4's dial-time pinning
// requirement), not just the pre-flight ValidateURL check: a request whose
// literal IP host is private/loopback must fail at dial time.
func Test_NewSSRFGuardedClient_RejectsDialToPrivateIP(t *testing.T) {
	client := NewSSRFGuardedClient(2 * time.Second)

	// 127.0.0.1 is loopback -- Control must reject the dial regardless of
	// whether anything is actually listening on the port.
	resp, err := client.Get("https://127.0.0.1:1/mcp")
	if err == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		t.Fatalf("expected the dial to a loopback address to be rejected, got a response")
	}
}

// Test_NewSSRFGuardedClient_WiringAllowsPublicLikeTraffic verifies the
// http.Client plumbing itself (transport, timeout, TLS config) is otherwise
// functional by pointing it at a local httptest.Server. httptest.Server
// binds to 127.0.0.1, which the SSRF guard correctly rejects as loopback --
// so this test asserts the guard fires (proving Control is wired into this
// exact client), while Test_rejectDisallowedIP_TableTest above is what
// verifies real public addresses are accepted by the classifier that same
// Control hook calls.
func Test_NewSSRFGuardedClient_WiringRejectsLocalTestServer(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewSSRFGuardedClient(2 * time.Second)
	resp, err := client.Get(srv.URL)
	if err == nil {
		if resp != nil {
			_ = resp.Body.Close()
		}
		t.Fatalf("expected the SSRF guard to reject a dial to the loopback-bound test server")
	}
}

// Test_NewSSRFGuardedClient_BoundsResponseHeaders pins the production
// client's response-header limit. Unit tests of the MCP client swap in a
// plain client, so without this the limit would be untested where it
// actually runs.
func Test_NewSSRFGuardedClient_BoundsResponseHeaders(t *testing.T) {
	transport, ok := NewSSRFGuardedClient(time.Second).Transport.(*httpsOnlyTransport)
	if !ok {
		t.Fatal("expected the https-only transport")
	}
	if transport.next.MaxResponseHeaderBytes != mcpMaxResponseHeaderBytes {
		t.Fatalf("MaxResponseHeaderBytes = %d, want %d", transport.next.MaxResponseHeaderBytes, mcpMaxResponseHeaderBytes)
	}

	// Exercise the limit through the production transport itself, with only
	// the loopback-rejecting dial hook removed so it can reach the test
	// server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", strings.Repeat("a", 200<<10))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	probe := transport.next.Clone()
	probe.DialContext = (&net.Dialer{Timeout: time.Second}).DialContext
	resp, err := (&http.Client{Transport: probe, Timeout: 2 * time.Second}).Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected a 200 KiB response header to be refused")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected the header limit error, got: %v", err)
	}
}

// loopbackGuardedClient is the production client with only the dial hook
// that rejects loopback removed, trusting srv's certificate, so tests reach a
// local TLS server through the same redirect and scheme policy.
func loopbackGuardedClient(srv *httptest.Server) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return newGuardedClient(2*time.Second, (&net.Dialer{Timeout: time.Second}).DialContext, &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool})
}

// Test_NewSSRFGuardedClient_RefusesRedirects pins that no redirect is
// followed, so neither a custom API-key header nor Authorization can be
// carried to another host or downgraded to http, and that the error names
// the target without its path or query.
func Test_NewSSRFGuardedClient_RefusesRedirects(t *testing.T) {
	var elsewhere atomic.Int32
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhere.Add(1)
	}))
	defer other.Close()

	tests := []struct {
		name        string
		location    func(self string) string
		wantHint    bool
		mustNotShow string
	}{
		{name: "another host", location: func(string) string { return other.URL + "/steal?token=secret-in-query" }, mustNotShow: "secret-in-query"},
		{name: "same host over http", location: func(self string) string { return strings.Replace(self, "https://", "http://", 1) + "/mcp" }},
		{name: "same path with a trailing slash removed", location: func(self string) string { return self + "/mcp" }, wantHint: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var srv *httptest.Server
			srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, tt.location(srv.URL), http.StatusTemporaryRedirect)
			}))
			defer srv.Close()

			req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp/", strings.NewReader(`{}`))
			req.Header.Set("X-API-Key", "api-key-value")
			req.Header.Set("Authorization", "Bearer bearer-value")
			resp, err := loopbackGuardedClient(srv).Do(req)
			if err == nil {
				_ = resp.Body.Close()
				t.Fatal("expected the redirect to be refused")
			}
			err = RedactTransportError(err)
			if !errors.Is(err, ErrRedirectRefused) {
				t.Fatalf("expected a refused redirect, got: %v", err)
			}
			if tt.wantHint != strings.Contains(err.Error(), "trailing slash") {
				t.Fatalf("trailing slash hint present=%v, want %v: %v", !tt.wantHint, tt.wantHint, err)
			}
			if tt.mustNotShow != "" && strings.Contains(err.Error(), tt.mustNotShow) {
				t.Fatalf("refusal exposes the redirect target's query: %v", err)
			}
			if strings.Contains(err.Error(), "/steal") || strings.Contains(err.Error(), "/mcp") {
				t.Fatalf("refusal exposes the redirect target's path: %v", err)
			}
		})
	}

	if n := elsewhere.Load(); n != 0 {
		t.Fatalf("the redirect target received %d request(s); no credential may leave the registered host", n)
	}
}

// Test_NewSSRFGuardedClient_RefusesPlainHTTP pins that a request that is
// not https is refused before anything is sent, even though stored URLs are
// only validated when they are written.
func Test_NewSSRFGuardedClient_RefusesPlainHTTP(t *testing.T) {
	var received atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
	}))
	defer srv.Close()

	tlsSrv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer tlsSrv.Close()

	resp, err := loopbackGuardedClient(tlsSrv).Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected a plain http request to be refused")
	}
	if !strings.Contains(err.Error(), "https only") {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := received.Load(); n != 0 {
		t.Fatalf("the plain http server received %d request(s)", n)
	}
}

// Test_NewSSRFGuardedClient_ClosesIdleConnections pins that wrapping the
// transport kept CloseIdleConnections working. http.Client reaches it only
// through an interface check, so a wrapper without it would compile and
// silently leak every call's idle connections.
func Test_NewSSRFGuardedClient_ClosesIdleConnections(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()

	client := loopbackGuardedClient(srv)
	if _, ok := client.Transport.(interface{ CloseIdleConnections() }); !ok {
		t.Fatal("the guarded transport does not forward CloseIdleConnections")
	}

	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		c := loopbackGuardedClient(srv)
		resp, err := c.Get(srv.URL)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		c.CloseIdleConnections()
	}
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > before+5 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if after := runtime.NumGoroutine(); after > before+5 {
		t.Fatalf("goroutines grew from %d to %d over 20 closed clients", before, after)
	}
}

// Test_NewSSRFGuardedClient_UnparseableLocationIsRedacted pins the redirect
// net/http cannot follow at all: a Location it cannot parse never reaches
// CheckRedirect, and net/http's own error quotes the value in full. The
// transport refuses it first, so the query never reaches an error.
func Test_NewSSRFGuardedClient_UnparseableLocationIsRedacted(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://bad host/%zz?token=SECRETPATH")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer srv.Close()

	resp, err := loopbackGuardedClient(srv).Post(srv.URL+"/mcp", "application/json", strings.NewReader(`{}`))
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected the redirect to be refused")
	}
	err = RedactTransportError(err)
	if !errors.Is(err, ErrRedirectRefused) {
		t.Fatalf("expected a refused redirect, got: %v", err)
	}
	if strings.Contains(err.Error(), "SECRETPATH") || strings.Contains(err.Error(), "%zz") {
		t.Fatalf("refusal quotes the unparseable location: %v", err)
	}
}

// Test_NewSSRFGuardedClient_RefusalBoundsTheHost pins that a server choosing
// a very long redirect host cannot make the error arbitrarily long.
func Test_NewSSRFGuardedClient_RefusalBoundsTheHost(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://"+strings.Repeat("a", 50000)+".example.com/x")
		w.WriteHeader(http.StatusFound)
	}))
	defer srv.Close()

	resp, err := loopbackGuardedClient(srv).Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("expected the redirect to be refused")
	}
	if msg := RedactTransportError(err).Error(); len(msg) > 512 {
		t.Fatalf("refusal is %d bytes; the server-chosen host must be cut", len(msg))
	}
}

// Test_RedactTransportError pins that a transport failure is reported
// without the request URL's path or query, where a customer may have put a
// key, while keeping the cause for errors.Is.
func Test_RedactTransportError(t *testing.T) {
	cause := errors.New("connection refused")
	err := RedactTransportError(&url.Error{Op: "Post", URL: "https://mcp.example.com/v1/mcp?api_key=SECRETQUERY", Err: cause})
	if strings.Contains(err.Error(), "SECRETQUERY") || strings.Contains(err.Error(), "/v1/mcp") {
		t.Fatalf("path or query survived: %v", err)
	}
	if !strings.Contains(err.Error(), "https://mcp.example.com") || !errors.Is(err, cause) {
		t.Fatalf("host or cause lost: %v", err)
	}

	plain := errors.New("not a url error")
	if RedactTransportError(plain) != plain {
		t.Fatal("an error without a URL must pass through unchanged")
	}
}

// Test_redirectRefusal_TrailingSlashHint pins when the trailing-slash hint is
// given.
func Test_redirectRefusal_TrailingSlashHint(t *testing.T) {
	mustURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("bad url %q: %v", s, err)
		}
		return u
	}
	tests := []struct {
		from, to string
		want     bool
	}{
		{"https://h/mcp/", "https://h/mcp", true},
		{"https://h/mcp", "https://h/mcp/", true},
		{"https://h/", "https://h", true},
		{"https://h/mcp", "https://h/other", false},
		{"https://h/mcp/", "http://h/mcp", false},
		{"https://h/mcp/", "https://other/mcp", false},
		{"https://h/mcp", "https://h/mcp%2F", false},
	}
	for _, tt := range tests {
		got := strings.Contains(redirectRefusal(mustURL(tt.from), mustURL(tt.to)).Error(), "trailing slash")
		if got != tt.want {
			t.Errorf("%s -> %s: hint %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
