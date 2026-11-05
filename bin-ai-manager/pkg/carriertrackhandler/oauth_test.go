package carriertrackhandler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"monorepo/bin-ai-manager/models/mcpserver"
)

func testCredential() mcpserver.Credential {
	return mcpserver.Credential{ID: "test-client-id", Secret: "test-client-secret"}
}

// noRedirectClient returns the test server's client with redirects left
// unfollowed, as the guarded production client does.
func noRedirectClient(srv *httptest.Server) *http.Client {
	c := srv.Client()
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return c
}

func newOAuthDeps(srv *httptest.Server, clk *fakeClock) Deps {
	return Deps{
		Carrier:  "fake",
		Client:   noRedirectClient(srv),
		Now:      clk.now,
		Tokens:   newTokenCache(128, clk.now),
		TokenKey: "token-key",
		MaxBody:  1 << 20,
	}
}

func Test_ExchangeClientCredentials_status(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		expectToken  string
		expectCode   ErrCode
		expectCool   time.Duration
		expectCached bool
	}{
		{"success", 200, `{"access_token":"tok1","token_type":"Bearer","expires_in":3600}`, "tok1", "", 0, true},
		{"token type is compared without case", 200, `{"access_token":"tok1","token_type":"bearer","expires_in":3600}`, "tok1", "", 0, true},
		{"token type absent is accepted", 200, `{"access_token":"tok1","expires_in":3600}`, "tok1", "", 0, true},
		{"token type other than bearer", 200, `{"access_token":"tok1","token_type":"mac","expires_in":3600}`, "", CodeCarrierUnavailable, 0, false},
		{"token with a line break", 200, `{"access_token":"tok\r\nX-Evil: 1","expires_in":3600}`, "", CodeCarrierUnavailable, 0, false},
		{"token with a space", 200, `{"access_token":"tok 1","expires_in":3600}`, "", CodeCarrierUnavailable, 0, false},
		{"empty token", 200, `{"access_token":"","token_type":"Bearer","expires_in":3600}`, "", CodeCarrierUnavailable, 0, false},
		{"broken json", 200, `{"access_token":`, "", CodeCarrierUnavailable, 0, false},
		{"not json", 200, `<html>`, "", CodeCarrierUnavailable, 0, false},
		{"expires_in absent: used once, not cached", 200, `{"access_token":"tok1","token_type":"Bearer"}`, "tok1", "", 0, false},
		{"expires_in zero: used once, not cached", 200, `{"access_token":"tok1","expires_in":0}`, "tok1", "", 0, false},
		{"expires_in as a string: used once, not cached", 200, `{"access_token":"tok1","expires_in":"3600"}`, "tok1", "", 0, false},
		{"expires_in below the margin: used once, not cached", 200, `{"access_token":"tok1","expires_in":30}`, "tok1", "", 0, false},
		{"expires_in absurdly large is capped, not overflowed", 200, `{"access_token":"tok1","expires_in":1e300}`, "tok1", "", 0, true},
		{"expires_in negative: used once, not cached", 200, `{"access_token":"tok1","expires_in":-5}`, "tok1", "", 0, false},
		{"token over 4 KiB", 200, `{"access_token":"` + strings.Repeat("a", 4097) + `","expires_in":3600}`, "", CodeCarrierUnavailable, 0, false},
		{"401 means the key was rejected", 401, `{"error":"invalid_client","error_description":"SECRET-DETAIL"}`, "", CodeCredentialInvalid, 0, false},
		{"400 is not blamed on the key", 400, `{}`, "", CodeCarrierUnavailable, 0, false},
		{"403 is a short block", 403, `{}`, "", CodeCarrierUnavailable, time.Minute, false},
		{"429 is busy without a cooldown", 429, `{}`, "", CodeServiceBusy, 0, false},
		{"500", 500, `{}`, "", CodeCarrierUnavailable, 0, false},
		{"503", 503, `{}`, "", CodeCarrierUnavailable, 0, false},
		{"redirect", 302, ``, "", CodeCarrierUnavailable, 0, false},
		{"other 4xx", 418, `{}`, "", CodeCarrierUnavailable, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.status == 302 {
					w.Header().Set("Location", "https://elsewhere.example.com/x")
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			clk := newFakeClock()
			d := newOAuthDeps(srv, clk)

			token, cerr := d.AccessToken(context.Background(), TokenSpec{URL: srv.URL + "/oauth/token"}, testCredential())
			if tt.expectCode != "" {
				if cerr == nil || cerr.Code != tt.expectCode || cerr.Cooldown != tt.expectCool {
					t.Fatalf("expected %s with cooldown %v, got %+v", tt.expectCode, tt.expectCool, cerr)
				}
				if token != "" {
					t.Errorf("no token may be returned with an error")
				}
				if _, ok := d.Tokens.Get("token-key"); ok {
					t.Errorf("nothing may be cached after an error")
				}
				return
			}
			if cerr != nil || token != tt.expectToken {
				t.Fatalf("got (%q, %+v), expected %q", token, cerr, tt.expectToken)
			}
			_, cached := d.Tokens.Get("token-key")
			if cached != tt.expectCached {
				t.Errorf("cached = %v, expected %v", cached, tt.expectCached)
			}
		})
	}
}

func Test_AccessToken_ttl(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn int
		liveAt    time.Duration
		goneAt    time.Duration
	}{
		{"one hour token lives for the hour minus the margin", 3600, 3500 * time.Second, 3601 * time.Second},
		{"a long token is cached for an hour at most", 86400, 3599 * time.Second, 3601 * time.Second},
		{"a short token", 120, 59 * time.Second, 61 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exchanges atomic.Int32
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				exchanges.Add(1)
				_, _ = fmt.Fprintf(w, `{"access_token":"tok","token_type":"Bearer","expires_in":%d}`, tt.expiresIn)
			}))
			defer srv.Close()

			clk := newFakeClock()
			d := newOAuthDeps(srv, clk)
			spec := TokenSpec{URL: srv.URL}

			for i := 0; i < 3; i++ {
				if _, cerr := d.AccessToken(context.Background(), spec, testCredential()); cerr != nil {
					t.Fatal(cerr)
				}
			}
			if exchanges.Load() != 1 {
				t.Errorf("a cached token must not be exchanged again: %d exchanges", exchanges.Load())
			}

			clk.add(tt.liveAt)
			_, _ = d.AccessToken(context.Background(), spec, testCredential())
			if exchanges.Load() != 1 {
				t.Errorf("the token must still be live at %v", tt.liveAt)
			}

			clk.add(tt.goneAt - tt.liveAt)
			_, _ = d.AccessToken(context.Background(), spec, testCredential())
			if exchanges.Load() != 2 {
				t.Errorf("the token must have expired at %v: %d exchanges", tt.goneAt, exchanges.Load())
			}
		})
	}
}

func Test_ExchangeClientCredentials_requestShape(t *testing.T) {
	type seen struct {
		method, contentType, auth, accept, extra, body string
	}
	run := func(t *testing.T, spec TokenSpec) seen {
		t.Helper()
		var got seen
		srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			got = seen{r.Method, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), r.Header.Get("Accept"), r.Header.Get("X-Extra"), string(b)}
			if r.URL.RawQuery != "" {
				t.Errorf("the credential must never be in the url query: %q", r.URL.RawQuery)
			}
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
		}))
		defer srv.Close()

		d := newOAuthDeps(srv, newFakeClock())
		spec.URL = srv.URL + "/token"
		if _, cerr := d.AccessToken(context.Background(), spec, testCredential()); cerr != nil {
			t.Fatal(cerr)
		}
		return got
	}

	basic := run(t, TokenSpec{ExtraHeaders: map[string]string{"X-Extra": "v1"}, ExtraForm: map[string]string{"scope": "track"}})
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("test-client-id:test-client-secret"))
	if basic.method != "POST" || basic.auth != wantAuth || basic.extra != "v1" || basic.accept != "application/json" ||
		!strings.HasPrefix(basic.contentType, "application/x-www-form-urlencoded") {
		t.Errorf("basic request wrong: %+v", basic)
	}
	if !strings.Contains(basic.body, "grant_type=client_credentials") || !strings.Contains(basic.body, "scope=track") ||
		strings.Contains(basic.body, "test-client-secret") {
		t.Errorf("basic body wrong: %q", basic.body)
	}

	form := run(t, TokenSpec{Style: TokenAuthFormBody})
	if form.auth != "" {
		t.Errorf("a form body request must not send basic auth: %q", form.auth)
	}
	for _, want := range []string{"grant_type=client_credentials", "client_id=test-client-id", "client_secret=test-client-secret"} {
		if !strings.Contains(form.body, want) {
			t.Errorf("form body %q does not contain %q", form.body, want)
		}
	}
}

func Test_ExchangeClientCredentials_mapOverride(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(400) }))
	defer srv.Close()
	d := newOAuthDeps(srv, newFakeClock())

	spec := TokenSpec{URL: srv.URL, Map: func(status int) *CarrierError {
		if status == 400 {
			return &CarrierError{Code: CodeCredentialInvalid}
		}
		return nil
	}}
	_, cerr := d.AccessToken(context.Background(), spec, testCredential())
	if cerr == nil || cerr.Code != CodeCredentialInvalid {
		t.Errorf("the carrier specific mapping must win: %+v", cerr)
	}
}

func Test_ExchangeClientCredentials_bodyBound(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"tok","pad":"`+strings.Repeat("x", 70<<10)+`","expires_in":3600}`)
	}))
	defer srv.Close()
	d := newOAuthDeps(srv, newFakeClock())

	_, cerr := d.AccessToken(context.Background(), TokenSpec{URL: srv.URL}, testCredential())
	if cerr == nil || cerr.Code != CodeCarrierUnavailable {
		t.Errorf("a token response over 64 KiB must be refused, got %+v", cerr)
	}
}

func Test_ExchangeClientCredentials_transportFailureHasNoDetail(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	d := newOAuthDeps(srv, newFakeClock())
	_, cerr := d.AccessToken(context.Background(), TokenSpec{URL: url + "/secret-path"}, testCredential())
	if cerr == nil || cerr.Code != CodeCarrierUnavailable {
		t.Errorf("got %+v", cerr)
	}
}

func Test_ExchangeClientCredentials_contextCancelled(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)

	d := newOAuthDeps(srv, newFakeClock())
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, cerr := d.AccessToken(ctx, TokenSpec{URL: srv.URL}, testCredential())
	if cerr == nil || cerr.Code != CodeCarrierUnavailable {
		t.Errorf("got %+v", cerr)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("the exchange must stop at the context deadline")
	}
}

func Test_AccessToken_noKeyMeansNoCaching(t *testing.T) {
	var exchanges atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		exchanges.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
	}))
	defer srv.Close()

	d := newOAuthDeps(srv, newFakeClock())
	d.TokenKey = ""
	for i := 0; i < 2; i++ {
		if _, cerr := d.AccessToken(context.Background(), TokenSpec{URL: srv.URL}, testCredential()); cerr != nil {
			t.Fatal(cerr)
		}
	}
	if exchanges.Load() != 2 {
		t.Errorf("without a token key every call exchanges: %d", exchanges.Load())
	}
}

func Test_ReadBody(t *testing.T) {
	if b, err := ReadBody(strings.NewReader("abcd"), 4); err != nil || string(b) != "abcd" {
		t.Errorf("exactly the limit must pass: %q %v", b, err)
	}
	if _, err := ReadBody(strings.NewReader("abcde"), 4); err == nil {
		t.Errorf("one byte over the limit must be refused, not cut")
	}
}

func Test_numberHelpers(t *testing.T) {
	if got := CleanNumber(" 1z 999-aa1 0123 "); got != "1Z999AA10123" {
		t.Errorf("CleanNumber = %q", got)
	}
	for n, ok := range map[string]bool{"ABCD1234": true, "ABC": false, strings.Repeat("A", 41): false, "ABCD-123": false, "ABCD123\u00e9": false, "": false} {
		if got := LooseNumber(n, 8, 40); got != ok {
			t.Errorf("LooseNumber(%q) = %v, expected %v", n, got, ok)
		}
	}
}
