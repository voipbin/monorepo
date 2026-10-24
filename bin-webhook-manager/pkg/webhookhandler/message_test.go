package webhookhandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// Test_computeSignature verifies the HMAC-SHA256 signature computed for the
// X-VoIPBIN-Signature header matches the standard hmac.New(sha256.New, secret)
// construction used elsewhere in the monorepo for verifying inbound signatures.
func Test_computeSignature(t *testing.T) {

	type test struct {
		name   string
		secret string
		body   []byte
	}

	tests := []test{
		{
			name:   "normal",
			secret: "test-secret",
			body:   []byte(`{"type":"call_updated","data":{"id":"test"}}`),
		},
		{
			name:   "empty body",
			secret: "test-secret",
			body:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := computeSignature(tt.secret, tt.body)

			mac := hmac.New(sha256.New, []byte(tt.secret))
			mac.Write(tt.body)
			expect := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			if res != expect {
				t.Errorf("Wrong match. expect: %s, got: %s", expect, res)
			}
		})
	}
}

// Test_computeSignature_differentSecretsDiffer ensures the signature is
// actually keyed by the secret -- a required property for receivers to be
// able to distinguish an authentic request from a forged one.
func Test_computeSignature_differentSecretsDiffer(t *testing.T) {
	body := []byte(`{"type":"call_updated"}`)

	sig1 := computeSignature("secret-a", body)
	sig2 := computeSignature("secret-b", body)

	if sig1 == sig2 {
		t.Errorf("Wrong match. expected different signatures for different secrets, got the same: %s", sig1)
	}
}

// Test_computeSignature_knownVector pins the signature to values computed
// independently (openssl dgst -sha256 -hmac), so a change of the algorithm or
// of the "sha256=" prefix is caught.
func Test_computeSignature_knownVector(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		expect string
	}{
		{
			name:   "empty body",
			body:   nil,
			expect: "sha256=a41bc6d81d6413576ae0994995e0ad89a416ec97389515c3604f47722122eeeb",
		},
		{
			name:   "json body",
			body:   []byte(`{"a":1}`),
			expect: "sha256=179bf20a8b9040a32368814a68b0dc270823b5968498e0a73796c4202708ed8d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := computeSignature("test-secret", tt.body); res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// stubTransport is a RoundTripper that records the requests it receives and
// answers from a scripted list, so no network is used.
type stubTransport struct {
	mu        sync.Mutex
	requests  []*http.Request
	bodies    []string
	responses []func() (*http.Response, error)
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var body string
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	s.requests = append(s.requests, req)
	s.bodies = append(s.bodies, body)

	// the last scripted response repeats.
	i := len(s.requests) - 1
	if i >= len(s.responses) {
		i = len(s.responses) - 1
	}
	return s.responses[i]()
}

func (s *stubTransport) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func stubStatus(code int) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	}
}

func stubError(msg string) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		return nil, errors.New(msg)
	}
}

func newStubHandler(st *stubTransport) *webhookHandler {
	return &webhookHandler{httpClient: &http.Client{Transport: st}}
}

// publicTestURI is a public IP literal: it passes the SSRF validation without a
// DNS lookup and the stub transport guarantees no connection is made.
const publicTestURI = "http://8.8.8.8/hook"

func Test_sendMessage_success(t *testing.T) {
	tests := []struct {
		name string

		method   string
		dataType string
		data     []byte
		secret   string

		expectContentType string
		expectSignature   string
	}{
		{
			name:              "signed json post",
			method:            "POST",
			dataType:          "application/json",
			data:              []byte(`{"a":1}`),
			secret:            "test-secret",
			expectContentType: "application/json",
			expectSignature:   "sha256=179bf20a8b9040a32368814a68b0dc270823b5968498e0a73796c4202708ed8d",
		},
		{
			name:     "no secret sends no signature",
			method:   "POST",
			dataType: "application/json",
			data:     []byte(`{"a":1}`),

			expectContentType: "application/json",
		},
		{
			name:   "no data type sends no content type",
			method: "PUT",
			data:   []byte(`{"a":1}`),
			secret: "test-secret",

			expectSignature: "sha256=179bf20a8b9040a32368814a68b0dc270823b5968498e0a73796c4202708ed8d",
		},
		{
			name:     "nil data is still signed",
			method:   "GET",
			dataType: "application/json",
			secret:   "test-secret",

			expectContentType: "",
			expectSignature:   "sha256=a41bc6d81d6413576ae0994995e0ad89a416ec97389515c3604f47722122eeeb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(200)}}
			h := newStubHandler(st)

			if err := h.sendMessage(context.Background(), publicTestURI, tt.method, tt.dataType, tt.data, tt.secret); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if st.calls() != 1 {
				t.Fatalf("Wrong match. expect 1 call, got: %d", st.calls())
			}
			req := st.requests[0]
			if req.Method != tt.method || req.URL.String() != publicTestURI {
				t.Errorf("Wrong match. got: %s %s", req.Method, req.URL)
			}
			if got := req.Header.Get("Content-Type"); got != tt.expectContentType {
				t.Errorf("Wrong match. expect content type: %q, got: %q", tt.expectContentType, got)
			}
			if got := req.Header.Get(signatureHeader); got != tt.expectSignature {
				t.Errorf("Wrong match. expect signature: %q, got: %q", tt.expectSignature, got)
			}
			if st.bodies[0] != string(tt.data) {
				t.Errorf("Wrong match. expect body: %q, got: %q", string(tt.data), st.bodies[0])
			}
		})
	}
}

// Test_sendMessage_clientError pins the current behaviour: a 4xx response is
// treated as delivered and is not retried.
func Test_sendMessage_clientError(t *testing.T) {
	for _, code := range []int{400, 404, 410, 429} {
		st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(code)}}
		h := newStubHandler(st)

		if err := h.sendMessage(context.Background(), publicTestURI, "POST", "application/json", []byte("{}"), ""); err != nil {
			t.Errorf("Wrong match. status: %d, expect: ok, got: %v", code, err)
		}
		if st.calls() != 1 {
			t.Errorf("Wrong match. status: %d, expect 1 call, got: %d", code, st.calls())
		}
	}
}

func Test_sendMessage_serverError_retries(t *testing.T) {
	t.Parallel()

	st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(503)}}
	h := newStubHandler(st)

	err := h.sendMessage(context.Background(), publicTestURI, "POST", "application/json", []byte("{}"), "")
	if err == nil || err.Error() != "server returned status 503" {
		t.Errorf("Wrong match. expect: server returned status 503, got: %v", err)
	}
	if st.calls() != 3 {
		t.Errorf("Wrong match. expect 3 attempts, got: %d", st.calls())
	}
}

func Test_sendMessage_serverError_thenSuccess(t *testing.T) {
	t.Parallel()

	st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(500), stubStatus(200)}}
	h := newStubHandler(st)

	if err := h.sendMessage(context.Background(), publicTestURI, "POST", "application/json", []byte("{}"), ""); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if st.calls() != 2 {
		t.Errorf("Wrong match. expect 2 attempts, got: %d", st.calls())
	}
}

func Test_sendMessage_transportError_retries(t *testing.T) {
	t.Parallel()

	st := &stubTransport{responses: []func() (*http.Response, error){stubError("boom")}}
	h := newStubHandler(st)

	err := h.sendMessage(context.Background(), publicTestURI, "POST", "application/json", []byte("{}"), "")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Wrong match. expect an error wrapping boom, got: %v", err)
	}
	if st.calls() != 3 {
		t.Errorf("Wrong match. expect 3 attempts, got: %d", st.calls())
	}
}

func Test_sendMessage_rejected(t *testing.T) {
	tests := []struct {
		name string

		uri    string
		method string
	}{
		{name: "empty uri", uri: "", method: "POST"},
		{name: "unsupported scheme", uri: "ftp://8.8.8.8/hook", method: "POST"},
		{name: "loopback", uri: "http://127.0.0.1/hook", method: "POST"},
		{name: "private network", uri: "http://10.0.0.5/hook", method: "POST"},
		{name: "link local metadata", uri: "http://169.254.169.254/latest", method: "POST"},
		{name: "invalid method", uri: publicTestURI, method: "BAD METHOD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &stubTransport{responses: []func() (*http.Response, error){stubStatus(200)}}
			h := newStubHandler(st)

			if err := h.sendMessage(context.Background(), tt.uri, tt.method, "application/json", []byte("{}"), ""); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if st.calls() != 0 {
				t.Errorf("Wrong match. expect no request to be sent, got: %d", st.calls())
			}
		})
	}
}
