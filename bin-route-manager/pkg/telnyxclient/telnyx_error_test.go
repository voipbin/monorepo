package telnyxclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type telnyxOp struct {
	name string
	call func(c TelnyxClient) error
}

func telnyxOps() []telnyxOp {
	ctx := context.Background()

	return []telnyxOp{
		{"validate key", func(c TelnyxClient) error { return c.ValidateKey(ctx) }},
		{"create voice profile", func(c TelnyxClient) error { _, err := c.CreateOutboundVoiceProfile(ctx, "n"); return err }},
		{"delete voice profile", func(c TelnyxClient) error { return c.DeleteOutboundVoiceProfile(ctx, "p1") }},
		{"create fqdn connection", func(c TelnyxClient) error {
			_, err := c.CreateFQDNConnection(ctx, "n", "p1", "user", "synthetic-credential")
			return err
		}},
		{"delete fqdn connection", func(c TelnyxClient) error { return c.DeleteFQDNConnection(ctx, "c1") }},
		{"register fqdn", func(c TelnyxClient) error {
			_, err := c.RegisterFQDN(ctx, "c1", "gw.example.invalid", 5060)
			return err
		}},
		{"delete fqdn", func(c TelnyxClient) error { return c.DeleteFQDN(ctx, "f1") }},
	}
}

func Test_operations_unexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)

	for _, op := range telnyxOps() {
		t.Run(op.name, func(t *testing.T) {
			err := op.call(c)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if errors.Is(err, ErrInvalidKey) {
				t.Errorf("Wrong match. expect: a generic status error, got: ErrInvalidKey")
			}
			if !strings.Contains(err.Error(), "500") {
				t.Errorf("Wrong match. expect: message with status 500, got: %v", err)
			}
		})
	}
}

func Test_operations_invalidKey(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))

		c := newTelnyxClientWithBase("testkey", srv.URL)

		// only the operations that map 401 and 403 to ErrInvalidKey
		ctx := context.Background()
		tests := []struct {
			name string
			call func() error
		}{
			{"validate key", func() error { return c.ValidateKey(ctx) }},
			{"create voice profile", func() error { _, err := c.CreateOutboundVoiceProfile(ctx, "n"); return err }},
			{"create fqdn connection", func() error {
				_, err := c.CreateFQDNConnection(ctx, "n", "p1", "user", "synthetic-credential")
				return err
			}},
			{"register fqdn", func() error { _, err := c.RegisterFQDN(ctx, "c1", "gw.example.invalid", 5060); return err }},
		}
		for _, tt := range tests {
			if err := tt.call(); !errors.Is(err, ErrInvalidKey) {
				t.Errorf("Wrong match. op: %s, status: %d, expect: ErrInvalidKey, got: %v", tt.name, status, err)
			}
		}

		srv.Close()
	}
}

func Test_operations_connectionFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := srv.URL
	srv.Close() // the port is closed, so every request is refused

	c := newTelnyxClientWithBase("testkey", baseURL)

	for _, op := range telnyxOps() {
		t.Run(op.name, func(t *testing.T) {
			err := op.call(c)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), "request failed") {
				t.Errorf("Wrong match. expect: message with request failed, got: %v", err)
			}
		})
	}
}

func Test_operations_invalidBaseURL(t *testing.T) {
	// a control character in the base url makes building the request fail before any network call
	c := newTelnyxClientWithBase("testkey", "http://gw.example.invalid/\x7f")

	for _, op := range telnyxOps() {
		t.Run(op.name, func(t *testing.T) {
			err := op.call(c)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), "build") {
				t.Errorf("Wrong match. expect: message with build, got: %v", err)
			}
		})
	}
}

func Test_operations_malformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{not-json")) //nolint:errcheck // test handler write to httptest recorder
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"create voice profile", func() error { _, err := c.CreateOutboundVoiceProfile(ctx, "n"); return err }},
		{"create fqdn connection", func() error {
			_, err := c.CreateFQDNConnection(ctx, "n", "p1", "user", "synthetic-credential")
			return err
		}},
		{"register fqdn", func() error { _, err := c.RegisterFQDN(ctx, "c1", "gw.example.invalid", 5060); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}
			if !strings.Contains(err.Error(), "decode") {
				t.Errorf("Wrong match. expect: message with decode, got: %v", err)
			}
		})
	}
}

func Test_connectionAndRegisterFQDN_statusBodyIncluded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("synthetic rejection reason")) //nolint:errcheck // test handler write to httptest recorder
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	ctx := context.Background()

	_, err := c.CreateFQDNConnection(ctx, "n", "p1", "user", "synthetic-credential")
	if err == nil || !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "synthetic rejection reason") {
		t.Errorf("Wrong match. expect: status and body in the error, got: %v", err)
	}

	_, err = c.RegisterFQDN(ctx, "c1", "gw.example.invalid", 5060)
	if err == nil || !strings.Contains(err.Error(), "422") || !strings.Contains(err.Error(), "synthetic rejection reason") {
		t.Errorf("Wrong match. expect: status and body in the error, got: %v", err)
	}
}

func Test_CreateOutboundVoiceProfile_retriesWithoutRejectedCountries(t *testing.T) {
	var mu sync.Mutex
	var attempts [][]string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Destinations []string `json:"whitelisted_destinations"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		attempts = append(attempts, req.Destinations)
		n := len(attempts)
		mu.Unlock()

		if n == 1 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"errors":[{"detail":"account not approved for the following countries: US, CA"}]}`)) //nolint:errcheck // test handler write to httptest recorder
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"profile-retry"}}`)) //nolint:errcheck // test handler write to httptest recorder
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)

	id, err := c.CreateOutboundVoiceProfile(context.Background(), "n")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if id != "profile-retry" {
		t.Errorf("Wrong match. expect: profile-retry, got: %s", id)
	}

	if len(attempts) != 2 {
		t.Fatalf("Wrong match. expect: 2 attempts, got: %d", len(attempts))
	}

	contains := func(list []string, v string) bool {
		for _, s := range list {
			if s == v {
				return true
			}
		}
		return false
	}
	if !contains(attempts[0], "US") || !contains(attempts[0], "CA") {
		t.Errorf("Wrong match. expect: first attempt includes US and CA")
	}
	if contains(attempts[1], "US") || contains(attempts[1], "CA") {
		t.Errorf("Wrong match. expect: retry excludes US and CA")
	}
	if len(attempts[1]) != len(attempts[0])-2 {
		t.Errorf("Wrong match. expect: %d destinations, got: %d", len(attempts[0])-2, len(attempts[1]))
	}
}

func Test_CreateOutboundVoiceProfile_unprocessable(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		expectSubstr string
	}{
		{"no country marker", `{"errors":[{"detail":"some other validation failure"}]}`, "unprocessable"},
		{"unparsable body", `plain text`, "unprocessable"},
		{"marker without valid codes", `{"errors":[{"detail":"the following countries: ABC"}]}`, "unprocessable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				w.Write([]byte(tt.body)) //nolint:errcheck // test handler write to httptest recorder
			}))
			defer srv.Close()

			c := newTelnyxClientWithBase("testkey", srv.URL)

			id, err := c.CreateOutboundVoiceProfile(context.Background(), "n")
			if err == nil || !strings.Contains(err.Error(), tt.expectSubstr) {
				t.Errorf("Wrong match. expect: error with %q, got: %v", tt.expectSubstr, err)
			}
			if id != "" {
				t.Errorf("Wrong match. expect: empty id, got: %s", id)
			}
		})
	}
}

func Test_CreateOutboundVoiceProfile_rejectedCountryNotInRequest(t *testing.T) {
	// Telnyx reports a country that was never requested, so retrying cannot make progress.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":[{"detail":"for the following countries: ZZ"}]}`)) //nolint:errcheck // test handler write to httptest recorder
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)

	_, err := c.CreateOutboundVoiceProfile(context.Background(), "n")
	if err == nil || !strings.Contains(err.Error(), "none could be identified") {
		t.Errorf("Wrong match. expect: none could be identified, got: %v", err)
	}
}

func Test_NewTelnyxClient_sendsBearerKey(t *testing.T) {
	c := NewTelnyxClient("synthetic-key")

	tc, ok := c.(*telnyxClient)
	if !ok {
		t.Fatalf("Wrong match. expect: *telnyxClient, got: %T", c)
	}
	if tc.baseURL != telnyxBaseURL {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", telnyxBaseURL, tc.baseURL)
	}

	// point the real client at a test server to confirm the key is sent as a bearer token
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tc.baseURL = srv.URL
	if err := tc.ValidateKey(context.Background()); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
