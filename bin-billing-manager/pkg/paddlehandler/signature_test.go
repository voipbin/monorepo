package paddlehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/internal/logtest"
	"monorepo/bin-billing-manager/models/account"
)

const (
	testSecret = "test-webhook-secret-not-real"
	testBody   = `{"event_id":"evt_01","event_type":"transaction.completed"}`
)

// sign builds a Paddle-Signature header value the way Paddle does.
func sign(secret string, ts int64, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + ":" + body))

	return fmt.Sprintf("ts=%d;h1=%s", ts, hex.EncodeToString(mac.Sum(nil)))
}

func Test_verifySignature(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	ts := now.Unix()

	tests := []struct {
		name string

		secret    string
		signature string
		body      string

		expectErr bool
	}{
		{name: "valid", secret: testSecret, signature: sign(testSecret, ts, testBody), body: testBody},
		{name: "valid, 300 seconds old (the edge)", secret: testSecret, signature: sign(testSecret, ts-300, testBody), body: testBody},
		{name: "valid, 300 seconds ahead (the edge)", secret: testSecret, signature: sign(testSecret, ts+300, testBody), body: testBody},
		{name: "wrong hash", secret: testSecret, signature: fmt.Sprintf("ts=%d;h1=%s", ts, strings.Repeat("0", 64)), body: testBody, expectErr: true},
		{name: "body changed", secret: testSecret, signature: sign(testSecret, ts, testBody), body: testBody + " ", expectErr: true},
		{name: "other secret", secret: "other-secret", signature: sign(testSecret, ts, testBody), body: testBody, expectErr: true},
		{name: "301 seconds old", secret: testSecret, signature: sign(testSecret, ts-301, testBody), body: testBody, expectErr: true},
		{name: "301 seconds ahead", secret: testSecret, signature: sign(testSecret, ts+301, testBody), body: testBody, expectErr: true},
		{name: "empty signature", secret: testSecret, signature: "", body: testBody, expectErr: true},
		{name: "no separator", secret: testSecret, signature: "ts=1700000000", body: testBody, expectErr: true},
		{name: "ts missing", secret: testSecret, signature: "h1=abcdef;x=1", body: testBody, expectErr: true},
		{name: "h1 missing", secret: testSecret, signature: "ts=1700000000;x=1", body: testBody, expectErr: true},
		{name: "ts is not a number", secret: testSecret, signature: "ts=abc;h1=abcdef", body: testBody, expectErr: true},
		{name: "empty ts", secret: testSecret, signature: "ts=;h1=abcdef", body: testBody, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifySignature(tt.secret, tt.signature, []byte(tt.body), now)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if err == nil {
				return
			}
			if !errors.Is(err, ErrSignatureInvalid) {
				t.Errorf("Wrong match. expect: ErrSignatureInvalid, got: %v", err)
			}
			if errors.Is(err, ErrKeyUnavailable) {
				t.Errorf("Wrong match. a rejected signature must not be reported as an unreadable key")
			}

			// the error names a category only: no signature, timestamp, body or secret
			text := err.Error()
			for _, leak := range []string{tt.secret, tt.body, tt.signature, "1700000000", "evt_01"} {
				if leak != "" && strings.Contains(text, leak) {
					t.Errorf("Wrong match. the error holds %q: %s", leak, text)
				}
			}
		})
	}
}

func newVerifyHandler(secret KeyFunc) *paddleHandler {
	return &paddleHandler{webhookSecret: secret, priceMap: map[string]account.PlanType{}}
}

// With no stored secret the webhook is rejected, whatever the signature is: the check fails closed.
func Test_VerifyWebhook_noKeyIsRejected(t *testing.T) {
	tests := []struct {
		name   string
		secret KeyFunc
	}{
		{name: "no function", secret: nil},
		{name: "empty value", secret: staticKey("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newVerifyHandler(tt.secret)

			// empty, garbage, valid in form and expired signatures are all rejected with the same error
			for _, signature := range []string{"", "garbage", sign(testSecret, time.Now().Unix(), testBody), sign(testSecret, time.Now().Unix()-3600, testBody)} {
				err := h.VerifyWebhook(context.Background(), signature, []byte(testBody))
				if !errors.Is(err, ErrCredentialNotConfigured) {
					t.Fatalf("Wrong match. expect: ErrCredentialNotConfigured, got: %v", err)
				}
				if errors.Is(err, ErrSignatureInvalid) || errors.Is(err, ErrKeyUnavailable) {
					t.Errorf("Wrong match. the error must not look like another kind of error: %v", err)
				}
			}
		})
	}
}

func Test_VerifyWebhook_withKey(t *testing.T) {
	h := newVerifyHandler(staticKey(testSecret))
	ctx := context.Background()
	now := time.Now().Unix()

	if err := h.VerifyWebhook(ctx, sign(testSecret, now, testBody), []byte(testBody)); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
	if err := h.VerifyWebhook(ctx, sign("other", now, testBody), []byte(testBody)); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("Wrong match. expect: ErrSignatureInvalid, got: %v", err)
	}
	if err := h.VerifyWebhook(ctx, sign(testSecret, now-301, testBody), []byte(testBody)); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("Wrong match. expect: ErrSignatureInvalid (expired), got: %v", err)
	}
	// a key exists, and the header is missing
	if err := h.VerifyWebhook(ctx, "", []byte(testBody)); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("Wrong match. expect: ErrSignatureInvalid (missing header), got: %v", err)
	}
}

// A key that can not be read is an error of its own kind. It is never "no key" and
// never a rejected signature.
func Test_VerifyWebhook_keyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "plain error", err: errors.New("database is down")},
		{name: "already wrapped", err: fmt.Errorf("%w: cannot decrypt", ErrKeyUnavailable)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newVerifyHandler(func(context.Context) (string, error) { return "", tt.err })

			// the signature is a valid one for the key that could not be read: the error must win
			err := h.VerifyWebhook(context.Background(), sign(testSecret, time.Now().Unix(), testBody), []byte(testBody))
			if !errors.Is(err, ErrKeyUnavailable) {
				t.Fatalf("Wrong match. expect: ErrKeyUnavailable, got: %v", err)
			}
			if errors.Is(err, ErrSignatureInvalid) {
				t.Errorf("Wrong match. an unreadable key must not look like a rejected signature")
			}
		})
	}
}

// The secret is resolved for every call: a rotation is used by the next call, and the handler keeps
// nothing.
func Test_VerifyWebhook_resolvedAtRequestTime(t *testing.T) {
	current := "key-a"
	h := newVerifyHandler(func(context.Context) (string, error) { return current, nil })
	ctx := context.Background()
	now := time.Now().Unix()

	if err := h.VerifyWebhook(ctx, sign("key-a", now, testBody), []byte(testBody)); err != nil {
		t.Fatalf("Wrong match. expect: nil for key A, got: %v", err)
	}

	current = "key-b"
	if err := h.VerifyWebhook(ctx, sign("key-a", now, testBody), []byte(testBody)); !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("Wrong match. expect: the signature of key A is rejected after the rotation, got: %v", err)
	}
	if err := h.VerifyWebhook(ctx, sign("key-b", now, testBody), []byte(testBody)); err != nil {
		t.Errorf("Wrong match. expect: nil for key B, got: %v", err)
	}

	// the secret disappears: the webhook is rejected
	current = ""
	if err := h.VerifyWebhook(ctx, sign("key-b", now, testBody), []byte(testBody)); !errors.Is(err, ErrCredentialNotConfigured) {
		t.Errorf("Wrong match. expect: ErrCredentialNotConfigured, got: %v", err)
	}
}

// The errors and the logs of the check hold the rejection category only: no signature, body, or
// secret.
func Test_VerifyWebhook_errorsAndLogsHoldNoSecret(t *testing.T) {
	logs := logtest.New(t)
	ctx := context.Background()

	// no key: the error is returned and the check logs nothing
	rejected := newVerifyHandler(nil)
	err := rejected.VerifyWebhook(ctx, "ts=1;h1=signature-text", []byte("body-text"))
	if !errors.Is(err, ErrCredentialNotConfigured) {
		t.Fatalf("Wrong match. expect: ErrCredentialNotConfigured, got: %v", err)
	}
	if logs.HasLevel(logrus.WarnLevel) {
		t.Errorf("Wrong match. expect: no warning from the check")
	}
	for _, text := range []string{"signature-text", "body-text"} {
		if strings.Contains(err.Error(), text) {
			t.Errorf("Wrong match. expect: no request text in the error, got: %v", err)
		}
	}

	// a rejected check: the error is returned, and nothing of the request is in it
	h := newVerifyHandler(staticKey(testSecret))
	err = h.VerifyWebhook(ctx, "ts=1;h1=signature-text", []byte("body-text"))
	for _, text := range []string{testSecret, "signature-text", "body-text"} {
		if err == nil || strings.Contains(err.Error(), text) {
			t.Errorf("Wrong match. expect: a rejection without request text, got: %v", err)
		}
	}
}

// A portal session without a stored API key stops before any request leaves.
func Test_CreatePortalSession_noKey(t *testing.T) {
	tests := []struct {
		name   string
		apiKey KeyFunc
	}{
		{name: "no function", apiKey: nil},
		{name: "empty value", apiKey: staticKey("")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(http.StatusCreated)
			}))
			defer server.Close()

			h := &paddleHandler{
				apiKey:     tt.apiKey,
				baseURL:    server.URL,
				httpClient: http.DefaultClient,
				priceMap:   map[string]account.PlanType{},
			}

			url, err := h.CreatePortalSession(context.Background(), "ctm_synthetic_001")
			if !errors.Is(err, ErrCredentialNotConfigured) || url != "" {
				t.Errorf("Wrong match. expect: ErrCredentialNotConfigured and no url, got: %q %v", url, err)
			}
			if errors.Is(err, ErrKeyUnavailable) {
				t.Errorf("Wrong match. a key that is not stored is not an unreadable key")
			}
			if requests != 0 {
				t.Errorf("Wrong match. expect: no request to Paddle, got: %d", requests)
			}
		})
	}
}

func Test_CreatePortalSession_keyResolvedPerCall(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"cpls_1","urls":{"general":{"overview":"https://portal.example.com/x"}}}}`))
	}))
	defer server.Close()

	current := "key-1"
	reads := 0
	h := &paddleHandler{
		apiKey:     func(context.Context) (string, error) { reads++; return current, nil },
		baseURL:    server.URL,
		httpClient: http.DefaultClient,
		priceMap:   map[string]account.PlanType{},
	}

	for _, key := range []string{"key-1", "key-2"} {
		current = key
		if _, err := h.CreatePortalSession(context.Background(), "ctm_synthetic_001"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	if reads != 2 || len(seen) != 2 || seen[0] != "Bearer key-1" || seen[1] != "Bearer key-2" {
		t.Errorf("Wrong match. expect: one read per call and the new key in the header, got: reads %d, headers %v", reads, seen)
	}
}

// A key that can not be read stops the call before any request leaves.
func Test_CreatePortalSession_keyError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	h := &paddleHandler{
		apiKey:     func(context.Context) (string, error) { return "", errors.New("database is down") },
		baseURL:    server.URL,
		httpClient: http.DefaultClient,
		priceMap:   map[string]account.PlanType{},
	}

	url, err := h.CreatePortalSession(context.Background(), "ctm_synthetic_001")
	if !errors.Is(err, ErrKeyUnavailable) || url != "" {
		t.Errorf("Wrong match. expect: ErrKeyUnavailable and no url, got: %q %v", url, err)
	}
	if requests != 0 {
		t.Errorf("Wrong match. expect: no request to Paddle, got: %d", requests)
	}
}
