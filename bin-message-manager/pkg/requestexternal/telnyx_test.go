package requestexternal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingReader struct{}

func (failingReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

// replaceDefaultTransport swaps http.DefaultTransport and restores it in the same call.
// The tests using it must not run in parallel.
func replaceDefaultTransport(t *testing.T, rt http.RoundTripper) {
	t.Helper()

	orig := http.DefaultTransport
	http.DefaultTransport = rt
	t.Cleanup(func() {
		http.DefaultTransport = orig
	})
}

func newResponse(status int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d", status),
		Header:     http.Header{},
		Body:       body,
	}
}

func Test_TelnyxSendMessage_request(t *testing.T) {
	var gotMethod, gotURL, gotAuth, gotContentType, gotAccept, gotBody string

	replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "api.telnyx.com" {
			t.Errorf("Unexpected host: %s", req.URL.Host)
			return nil, errors.New("unexpected host")
		}

		b, _ := io.ReadAll(req.Body)
		gotMethod = req.Method
		gotURL = req.URL.String()
		gotAuth = req.Header.Get("Authorization")
		gotContentType = req.Header.Get("Content-Type")
		gotAccept = req.Header.Get("Accept")
		gotBody = string(b)

		return newResponse(200, io.NopCloser(strings.NewReader(`{"data":{"id":"fake-message-id","record_type":"message","type":"SMS"}}`))), nil
	}))

	h := NewRequestExternal("fake-token-mb", "fake-token-tn")

	res, err := h.TelnyxSendMessage(context.Background(), "+15550100001", "+15550100002", "hello")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("Wrong method. expect: %s, got: %s", http.MethodPost, gotMethod)
	}
	if gotURL != "https://api.telnyx.com/v2/messages" {
		t.Errorf("Wrong URL. got: %s", gotURL)
	}
	if gotAuth != "Bearer fake-token-tn" {
		t.Errorf("Wrong authorization header. expect the telnyx token, got: %s", gotAuth)
	}
	if gotContentType != "application/json" || gotAccept != "application/json" {
		t.Errorf("Wrong content headers. content-type: %s, accept: %s", gotContentType, gotAccept)
	}
	expectBody := `{"from":"+15550100001","to":"+15550100002","text":"hello"}`
	if gotBody != expectBody {
		t.Errorf("Wrong body.\nexpect: %s\ngot: %s", expectBody, gotBody)
	}

	if res.Data.ID != "fake-message-id" || res.Data.RecordType != "message" || res.Data.Type != "SMS" {
		t.Errorf("Wrong response. got: %+v", res.Data)
	}
}

func Test_TelnyxSendMessage_status(t *testing.T) {
	tests := []struct {
		name string

		status int
		body   string

		expectErr bool
	}{
		{name: "199 is rejected", status: 199, body: `{}`, expectErr: true},
		{name: "200 is accepted", status: 200, body: `{}`, expectErr: false},
		{name: "299 is accepted", status: 299, body: `{}`, expectErr: false},
		{name: "300 is rejected", status: 300, body: `{}`, expectErr: true},
		{name: "404 is rejected", status: 404, body: `{"errors":[{"detail":"fake not found"}]}`, expectErr: true},
		{name: "500 is rejected", status: 500, body: `{"errors":[{"detail":"fake failure"}]}`, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return newResponse(tt.status, io.NopCloser(strings.NewReader(tt.body))), nil
			}))

			h := NewRequestExternal("fake-token-mb", "fake-token-tn")

			res, err := h.TelnyxSendMessage(context.Background(), "+15550100001", "+15550100002", "hello")
			if !tt.expectErr {
				if err != nil || res == nil {
					t.Errorf("Wrong match. expect: ok, got: res: %v, err: %v", res, err)
				}
				return
			}

			if err == nil || res != nil {
				t.Fatalf("Wrong match. expect: error, got: res: %v, err: %v", res, err)
			}
			expectMsg := fmt.Sprintf("API request failed with status code %d and body: %s", tt.status, tt.body)
			if err.Error() != expectMsg {
				t.Errorf("Wrong error.\nexpect: %s\ngot: %s", expectMsg, err.Error())
			}
		})
	}
}

func Test_TelnyxSendMessage_responseErrors(t *testing.T) {
	tests := []struct {
		name string

		roundTrip roundTripFunc

		expectContains string
	}{
		{
			name: "malformed json body on success status",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return newResponse(200, io.NopCloser(strings.NewReader(`{bad`))), nil
			},
			expectContains: "error unmarshalling JSON",
		},
		{
			name: "body read failure",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return newResponse(200, io.NopCloser(failingReader{})), nil
			},
			expectContains: "error reading response body",
		},
		{
			name: "transport failure",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("simulated transport failure")
			},
			expectContains: "error making request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceDefaultTransport(t, tt.roundTrip)

			h := NewRequestExternal("fake-token-mb", "fake-token-tn")

			res, err := h.TelnyxSendMessage(context.Background(), "+15550100001", "+15550100002", "hello")
			if err == nil || res != nil {
				t.Fatalf("Wrong match. expect: error, got: res: %v, err: %v", res, err)
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("Wrong error. expect it to contain %q, got: %v", tt.expectContains, err)
			}
		})
	}
}

func Test_TelnyxSendMessage_payload(t *testing.T) {
	var gotBody string

	replaceDefaultTransport(t, roundTripFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		gotBody = string(b)
		return newResponse(200, io.NopCloser(strings.NewReader(`{}`))), nil
	}))

	h := NewRequestExternal("fake-token-mb", "fake-token-tn")

	// characters that need JSON escaping must survive the round trip
	text := "line1\n\"quoted\" \u00e9"
	if _, err := h.TelnyxSendMessage(context.Background(), "+15550100001", "+15550100002", text); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := `{"from":"+15550100001","to":"+15550100002","text":"line1\n\"quoted\" é"}`
	if gotBody != expect {
		t.Errorf("Wrong body.\nexpect: %s\ngot: %s", expect, gotBody)
	}
}
