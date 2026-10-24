package webhookhandler

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func Test_newSafeHTTPClient_DialContext_blocked(t *testing.T) {
	client := newSafeHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Wrong match. expect: *http.Transport, got: %T", client.Transport)
	}

	tests := []struct {
		name string

		addr string

		expectErr string
	}{
		{name: "loopback", addr: "127.0.0.1:80", expectErr: "private/reserved IP blocked"},
		{name: "private network", addr: "10.0.0.5:443", expectErr: "private/reserved IP blocked"},
		{name: "ipv6 loopback", addr: "[::1]:80", expectErr: "private/reserved IP blocked"},
		{name: "link local metadata", addr: "169.254.169.254:80", expectErr: "private/reserved IP blocked"},
		{name: "address without port", addr: "nohostport", expectErr: "invalid address"},
		{name: "empty host cannot be resolved", addr: ":80", expectErr: "cannot resolve"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := transport.DialContext(context.Background(), "tcp", tt.addr)
			if err == nil {
				_ = conn.Close()
				t.Fatalf("Wrong match. expect: error, got: connection")
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_newSafeHTTPClient_CheckRedirect(t *testing.T) {
	client := newSafeHTTPClient()

	newReq := func(url string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("Could not create the request. err: %v", err)
		}
		return req
	}
	via := func(n int) []*http.Request {
		res := make([]*http.Request, n)
		for i := range res {
			res[i] = newReq("http://8.8.8.8/")
		}
		return res
	}

	tests := []struct {
		name string

		target string
		via    int

		expectErr string
	}{
		{name: "public target within the limit", target: "http://8.8.8.8/next", via: 2},
		{name: "third hop is refused", target: "http://8.8.8.8/next", via: 3, expectErr: "too many redirects"},
		{name: "private target is refused", target: "http://10.0.0.5/next", via: 1, expectErr: "redirect target blocked"},
		{name: "loopback target is refused", target: "http://127.0.0.1/next", via: 1, expectErr: "redirect target blocked"},
		{name: "unsupported scheme is refused", target: "ftp://8.8.8.8/next", via: 1, expectErr: "redirect target blocked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CheckRedirect(newReq(tt.target), via(tt.via))
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}
