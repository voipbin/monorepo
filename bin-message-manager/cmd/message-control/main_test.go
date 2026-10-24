package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"monorepo/bin-message-manager/internal/config"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// Test_newRequestExternal_tokenOrder guards the argument order of the provider tokens.
// It replaces http.DefaultTransport, so it must not run in parallel.
func Test_newRequestExternal_tokenOrder(t *testing.T) {
	orig := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = orig
	})

	var gotAuth string
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.telnyx.com" {
			t.Errorf("Unexpected host: %s", req.URL.Host)
			return nil, errors.New("unexpected host")
		}

		gotAuth = req.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200",
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(`{"data":{}}`)),
		}, nil
	})

	cfg := &config.Config{
		AuthtokenMessagebird: "fake-mb",
		AuthtokenTelnyx:      "fake-tn",
	}

	h := newRequestExternal(cfg)
	if _, err := h.TelnyxSendMessage(context.Background(), "+15550000001", "+15550000002", "hello"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if gotAuth != "Bearer fake-tn" {
		t.Errorf("Wrong authorization header. expect: Bearer fake-tn, got: %s", gotAuth)
	}
}
