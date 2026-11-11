package requestexternal

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// staticToken returns a TokenFunc that always hands out the same fake token.
func staticToken(s string) TokenFunc {
	return func(context.Context) (string, error) { return s, nil }
}

var errTestToken = errors.New("the token could not be read")

func failingToken(context.Context) (string, error) { return "", errTestToken }

// recordingTransport replaces the default transport for a test: it records the Authorization header
// of every request and answers 500 without any network use.
type recordingTransport struct {
	mu   sync.Mutex
	auth []string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.auth = append(r.auth, req.Header.Get("Authorization"))
	r.mu.Unlock()

	return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody, Header: http.Header{}, Request: req}, nil
}

// useRecordingTransport routes every send to a recorder, with no network use. The messagebird sms
// builds its own client, so its endpoint is pointed to a local server that records the same header.
func useRecordingTransport(t *testing.T) *recordingTransport {
	t.Helper()

	rec := &recordingTransport{}
	old := http.DefaultTransport
	http.DefaultTransport = rec
	t.Cleanup(func() { http.DefaultTransport = old })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = rec.RoundTrip(r)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	oldURI := messagebirdMessagesURI
	messagebirdMessagesURI = srv.URL
	t.Cleanup(func() { messagebirdMessagesURI = oldURI })

	return rec
}

// sendAll calls the four send methods once.
func sendAll(h RequestExternal) []error {
	ctx := context.Background()
	var errs []error

	_, err := h.TelnyxSendMessage(ctx, "+155****0001", "+155****0002", "hello")
	errs = append(errs, err)
	_, err = h.TelnyxSendMMS(ctx, "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"})
	errs = append(errs, err)
	_, err = h.MessagebirdSendMessage(ctx, "+155****0001", []string{"+155****0002"}, "hello")
	errs = append(errs, err)
	_, err = h.MessagebirdSendMMS(ctx, "+155****0001", "+155****0002", "hello", []string{"https://example.invalid/a.png"})
	errs = append(errs, err)

	return errs
}

// A token that can not be read stops the request before any connection is made, for every send
// method, and the error is handed to the caller (it is not turned into an empty token).
func Test_send_tokenUnavailable(t *testing.T) {
	rec := useRecordingTransport(t)

	names := []string{"telnyx sms", "telnyx mms", "messagebird sms", "messagebird mms"}
	for i, err := range sendAll(NewRequestExternal(failingToken, failingToken)) {
		if !errors.Is(err, errTestToken) {
			t.Errorf("Wrong match. %s expect: the token error, got: %v", names[i], err)
		}
		if !IsTokenUnavailable(err) {
			t.Errorf("Wrong match. %s: expect ErrTokenUnavailable, got: %v", names[i], err)
		}
		if IsDefinitiveRejection(err) {
			t.Errorf("Wrong match. %s: a token error must not be a definitive rejection", names[i])
		}
	}

	if len(rec.auth) != 0 {
		t.Errorf("Wrong match. expect: no request is sent, got: %d", len(rec.auth))
	}
}

// Only the token of the failing provider stops its requests: the other provider is not affected.
func Test_send_tokenUnavailableIsPerProvider(t *testing.T) {
	rec := useRecordingTransport(t)

	errs := sendAll(NewRequestExternal(staticToken("mb-ok"), failingToken))
	if !errors.Is(errs[0], errTestToken) || !errors.Is(errs[1], errTestToken) || errors.Is(errs[2], errTestToken) || errors.Is(errs[3], errTestToken) {
		t.Errorf("Wrong match. expect: only the telnyx sends fail on the token, got: %v", errs)
	}
	if strings.Join(rec.auth, "|") != "AccessKey mb-ok|AccessKey mb-ok" {
		t.Errorf("Wrong match. expect: two messagebird requests, got: %v", rec.auth)
	}
}

// The token is asked for on every request: a change between two sends is used by the second
// one, for each provider, and each provider uses its own token.
func Test_send_tokenIsReadPerRequest(t *testing.T) {
	rec := useRecordingTransport(t)

	mb, tn := "mb-1", "tn-1"
	h := NewRequestExternal(
		func(context.Context) (string, error) { return mb, nil },
		func(context.Context) (string, error) { return tn, nil },
	)

	sendAll(h)
	mb, tn = "mb-2", "tn-2"
	sendAll(h)

	expect := []string{
		"Bearer tn-1", "Bearer tn-1", "AccessKey mb-1", "AccessKey mb-1",
		"Bearer tn-2", "Bearer tn-2", "AccessKey mb-2", "AccessKey mb-2",
	}
	if strings.Join(rec.auth, "|") != strings.Join(expect, "|") {
		t.Errorf("Wrong match. expect: %v, got: %v", expect, rec.auth)
	}
}
