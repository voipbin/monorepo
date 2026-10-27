package mcpserverhandler

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func Test_ValidateURL_malformed(t *testing.T) {
	tests := []struct {
		name string
		raw  string

		expectMsg string
	}{
		{name: "unparseable url", raw: "https://[::1", expectMsg: "invalid url"},
		{name: "https without host", raw: "https:///only/path", expectMsg: "url has no host"},
		{name: "empty string", raw: "", expectMsg: "url scheme must be https"},
		{name: "link local literal", raw: "https://169.254.169.254/latest", expectMsg: ""},
		{name: "ipv4 mapped ipv6 literal", raw: "https://[::ffff:10.0.0.1]/", expectMsg: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.raw)
			if err == nil {
				t.Fatalf("expected error for %q", tt.raw)
			}
			if tt.expectMsg != "" && !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %q, expect to contain %q", err.Error(), tt.expectMsg)
			}
		})
	}
}

func Test_RefuseRedirects(t *testing.T) {
	mustURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		if err != nil {
			t.Fatalf("bad url %q: %v", s, err)
		}
		return u
	}

	tests := []struct {
		name string
		req  *http.Request
		via  []*http.Request

		expectHint bool
		expectHost string
	}{
		{
			name:       "with original request and slash difference",
			req:        &http.Request{URL: mustURL("https://h.example.com/mcp/")},
			via:        []*http.Request{{URL: mustURL("https://h.example.com/mcp")}},
			expectHint: true,
			expectHost: "https://h.example.com/mcp/"[:len("https://h.example.com")],
		},
		{
			name:       "other host gives no hint",
			req:        &http.Request{URL: mustURL("https://other.example.com/mcp")},
			via:        []*http.Request{{URL: mustURL("https://h.example.com/mcp")}},
			expectHost: "https://other.example.com",
		},
		{
			name:       "no previous request gives no hint",
			req:        &http.Request{URL: mustURL("https://h.example.com/mcp/")},
			expectHost: "https://h.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RefuseRedirects(tt.req, tt.via)
			if !errors.Is(err, ErrRedirectRefused) {
				t.Fatalf("expected ErrRedirectRefused, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.expectHost) {
				t.Errorf("error %q does not name %q", err.Error(), tt.expectHost)
			}
			if got := strings.Contains(err.Error(), "trailing slash"); got != tt.expectHint {
				t.Errorf("hint = %v, expect: %v", got, tt.expectHint)
			}
			if strings.Contains(err.Error(), "/mcp") {
				t.Errorf("path leaked into error: %q", err.Error())
			}
		})
	}
}

func Test_redirectRefusal_nilTarget(t *testing.T) {
	err := redirectRefusal(nil, nil)
	if !errors.Is(err, ErrRedirectRefused) || !strings.Contains(err.Error(), "could not be parsed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func Test_capText(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		n      int
		expect string
	}{
		{name: "shorter than limit", in: "abc", n: 5, expect: "abc"},
		{name: "exactly the limit", in: "abcde", n: 5, expect: "abcde"},
		{name: "cut ascii", in: "abcdefg", n: 3, expect: "abc..."},
		// each hangul syllable is 3 bytes; a cut at byte 4 falls inside the second one
		{name: "cut backs up to a rune boundary", in: "가나다", n: 4, expect: "가..."},
		{name: "cut on a rune boundary", in: "가나다", n: 6, expect: "가나..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := capText(tt.in, tt.n); got != tt.expect {
				t.Errorf("capText(%q, %d) = %q, expect: %q", tt.in, tt.n, got, tt.expect)
			}
		})
	}
}

type closeTrackingBody struct {
	closed bool
}

func (b *closeTrackingBody) Read([]byte) (int, error) { return 0, io.EOF }
func (b *closeTrackingBody) Close() error             { b.closed = true; return nil }

func Test_httpsOnlyTransport_RoundTrip_refusesNonHTTPS(t *testing.T) {
	body := &closeTrackingBody{}
	req, err := http.NewRequest(http.MethodPost, "http://mcp.example.com/rpc", body)
	if err != nil {
		t.Fatal(err)
	}

	// the wrapped transport is never reached, so it can be nil
	res, err := (&httpsOnlyTransport{}).RoundTrip(req)
	if res != nil {
		t.Errorf("expected nil response, got %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), `refusing a "http" request`) {
		t.Errorf("error = %v", err)
	}
	if !body.closed {
		t.Errorf("request body must be closed on refusal")
	}
}

func Test_controlRejectDisallowedAddr(t *testing.T) {
	tests := []struct {
		name    string
		address string

		expectMsg string // empty means allowed
	}{
		{name: "public literal", address: "203.0.113.10:443"},
		{name: "public ipv6 literal", address: "[2606:2800:220:1:248:1893:25c8:1946]:443"},
		{name: "no port", address: "203.0.113.10", expectMsg: "could not parse dial address"},
		{name: "hostname instead of ip", address: "mcp.example.com:443", expectMsg: "did not resolve to a literal IP"},
		{name: "loopback", address: "127.0.0.1:443", expectMsg: "rejected dial to 127.0.0.1:443 (tcp)"},
		{name: "private range", address: "10.1.2.3:443", expectMsg: "rejected dial"},
		{name: "link local metadata address", address: "169.254.169.254:80", expectMsg: "rejected dial"},
		{name: "ipv4 mapped private", address: "[::ffff:192.168.1.1]:443", expectMsg: "rejected dial"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controlRejectDisallowedAddr("tcp", tt.address, nil)
			if tt.expectMsg == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectMsg) {
				t.Errorf("error = %v, expect to contain %q", err, tt.expectMsg)
			}
		})
	}
}
