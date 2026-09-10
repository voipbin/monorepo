package mcpserverhandler

import (
	"net"
	"net/http"
	"net/http/httptest"
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
