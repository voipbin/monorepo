package emailhandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mailgun/mailgun-go/v4"
	"github.com/sendgrid/sendgrid-go"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-email-manager/pkg/dbhandler"
)

var errSupplier = errors.New("supplier failed")

// rotatingSecret returns the current value of v and counts the calls.
func rotatingSecret(v *atomic.Value, calls *atomic.Int32) SecretFunc {
	return func(context.Context) (string, error) {
		calls.Add(1)
		s, _ := v.Load().(string)
		return s, nil
	}
}

func failingSecret(calls *atomic.Int32) SecretFunc {
	return func(context.Context) (string, error) {
		calls.Add(1)
		return "", errSupplier
	}
}

func Test_engineMailgun_keyIsReadAtTheTimeOfTheSend(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	capture := &mailgunCapture{}
	server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
	defer server.Close()

	var keyValue atomic.Value
	var calls atomic.Int32
	keyValue.Store("first-synthetic-key")

	var seen []string
	h := &engineMailgun{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		apiKey:     rotatingSecret(&keyValue, &calls),
		newClient: func(apiKey string) *mailgun.MailgunImpl {
			seen = append(seen, apiKey)
			client := mailgun.NewMailgun(defaultMailgunDomain, apiKey)
			client.SetAPIBase(server.URL + "/v3")
			client.SetClient(&http.Client{Transport: loopbackOnlyTransport{}})
			return client
		},
	}

	if _, err := h.Send(context.Background(), testMailgunEmail(nil)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	keyValue.Store("rotated-synthetic-key")
	if _, err := h.Send(context.Background(), testMailgunEmail(nil)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(seen) != 2 || seen[0] != "first-synthetic-key" || seen[1] != "rotated-synthetic-key" {
		t.Errorf("Wrong match. expect: the key of each send, got: %v", seen)
	}
	if calls.Load() != 2 {
		t.Errorf("Wrong match. expect: 2 supplier calls, got: %d", calls.Load())
	}
}

// An empty key behaves as before: the client is built with the empty key and the send is attempted.
func Test_engineMailgun_emptyKeyBuildsTheClientAsBefore(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	capture := &mailgunCapture{}
	server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
	defer server.Close()

	var seen []string
	h := &engineMailgun{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		apiKey:     staticSecret(""),
		newClient: func(apiKey string) *mailgun.MailgunImpl {
			seen = append(seen, apiKey)
			client := mailgun.NewMailgun(defaultMailgunDomain, apiKey)
			client.SetAPIBase(server.URL + "/v3")
			client.SetClient(&http.Client{Transport: loopbackOnlyTransport{}})
			return client
		},
	}

	// the mailgun library itself refuses to send without a key, exactly as it did when the client
	// was built once at construction.
	_, err := h.Send(context.Background(), testMailgunEmail(nil))
	if err == nil || !strings.Contains(err.Error(), "valid api-key") {
		t.Errorf("Wrong match. expect: the library refusal, got: %v", err)
	}
	if len(seen) != 1 || seen[0] != "" || capture.count() != 0 {
		t.Errorf("Wrong match. expect: one client with the empty key and no request, got: %q, %d", seen, capture.count())
	}
}

func Test_engineMailgun_supplierErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var calls atomic.Int32
	built := false
	h := &engineMailgun{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		apiKey:     failingSecret(&calls),
		newClient:  func(string) *mailgun.MailgunImpl { built = true; return nil },
	}
	if _, err := h.Send(context.Background(), testMailgunEmail(nil)); err == nil || !errors.Is(err, errSupplier) {
		t.Errorf("Wrong match. expect: the supplier error, got: %v", err)
	}
	if built {
		t.Errorf("Wrong match. a client was built after a supplier error (the key must not turn into an empty key)")
	}

	h2 := &engineMailgun{reqHandler: requesthandler.NewMockRequestHandler(mc)}
	if _, err := h2.Send(context.Background(), testMailgunEmail(nil)); err == nil {
		t.Errorf("Wrong match. expect: error without a supplier, got: nil")
	}
}

func Test_engineMailgun_defaultClientConstructor(t *testing.T) {
	h := NewEngineMailgun(nil, staticSecret("synthetic-key")).(*engineMailgun)
	client, err := h.getClient(context.Background())
	if err != nil || client == nil || client.APIKey() != "synthetic-key" || client.Domain() != defaultMailgunDomain {
		t.Errorf("Wrong match. got: %v %v", client, err)
	}
}

func Test_engineSendgrid_keyIsReadAtTheTimeOfTheSend(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var auth []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("X-Message-Id", "synthetic-message-id")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	var keyValue atomic.Value
	var calls atomic.Int32
	keyValue.Store("first-synthetic-key")

	h := &engineSendgrid{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		apiKey:     rotatingSecret(&keyValue, &calls),
		newClient: func(apiKey string) *sendgrid.Client {
			client := sendgrid.NewSendClient(apiKey)
			client.BaseURL = server.URL + "/v3/mail/send"
			return client
		},
	}

	if _, err := h.Send(context.Background(), testSendgridEmail(nil)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	keyValue.Store("rotated-synthetic-key")
	if _, err := h.Send(context.Background(), testSendgridEmail(nil)); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if len(auth) != 2 || auth[0] != "Bearer first-synthetic-key" || auth[1] != "Bearer rotated-synthetic-key" {
		t.Errorf("Wrong match. expect: the key of each send, got: %v", auth)
	}
}

func Test_engineSendgrid_supplierErrors(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var calls atomic.Int32
	h := &engineSendgrid{reqHandler: requesthandler.NewMockRequestHandler(mc), apiKey: failingSecret(&calls)}
	if _, err := h.Send(context.Background(), testSendgridEmail(nil)); err == nil || !errors.Is(err, errSupplier) {
		t.Errorf("Wrong match. expect: the supplier error, got: %v", err)
	}

	h2 := &engineSendgrid{reqHandler: requesthandler.NewMockRequestHandler(mc)}
	if _, err := h2.Send(context.Background(), testSendgridEmail(nil)); err == nil {
		t.Errorf("Wrong match. expect: error without a supplier, got: nil")
	}

	h3 := NewEngineSendgrid(nil, staticSecret("synthetic-key")).(*engineSendgrid)
	client, err := h3.getClient(context.Background())
	if err != nil || client == nil {
		t.Errorf("Wrong match. got: %v %v", client, err)
	}
}

// The webhook signature: malformed requests cause no key lookup, the key is read right before the
// HMAC, a failing or missing or empty supplier rejects the request.
func Test_hookMailgun_signatureKeyLookup(t *testing.T) {
	vars := `{"voipbin_message_id":"` + testMailgunEmailID + `"}`

	tests := []struct {
		name string

		sig      string
		supplier func(calls *atomic.Int32) SecretFunc

		expectCalls int32
	}{
		{name: "missing signature causes no lookup", sig: "", supplier: func(c *atomic.Int32) SecretFunc { return countingSecret(testMailgunSigningKey, c) }, expectCalls: 0},
		{name: "incomplete signature causes no lookup", sig: mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, "", ""), supplier: func(c *atomic.Int32) SecretFunc { return countingSecret(testMailgunSigningKey, c) }, expectCalls: 0},
		{name: "bad timestamp causes no lookup", sig: mailgunTestSignatureWith(`"abc"`, testMailgunToken, "00"), supplier: func(c *atomic.Int32) SecretFunc { return countingSecret(testMailgunSigningKey, c) }, expectCalls: 0},
		{name: "not hex causes no lookup", sig: mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, "zz"), supplier: func(c *atomic.Int32) SecretFunc { return countingSecret(testMailgunSigningKey, c) }, expectCalls: 0},
		{name: "well formed signature reads the key once", sig: mailgunTestSignatureWith(`"`+testMailgunTimestamp+`"`, testMailgunToken, "aa"), supplier: func(c *atomic.Int32) SecretFunc { return countingSecret(testMailgunSigningKey, c) }, expectCalls: 1},
		{name: "supplier error rejects", sig: mailgunTestSignature(), supplier: failingSecret, expectCalls: 1},
		{name: "empty key rejects", sig: mailgunTestSignature(), supplier: func(c *atomic.Int32) SecretFunc { return countingSecret("", c) }, expectCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			var calls atomic.Int32
			// strict mocks: any db call or webhook publish fails the test.
			h := &emailHandler{
				db:                       dbhandler.NewMockDBHandler(mc),
				notifyHandler:            notifyhandler.NewMockNotifyHandler(mc),
				mailgunWebhookSigningKey: tt.supplier(&calls),
				timeNow:                  testMailgunNow,
			}

			data := mailgunTestPayloadSigned("delivered", "", vars, true, tt.sig)
			if err := h.hookMailgun(context.Background(), data); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if calls.Load() != tt.expectCalls {
				t.Errorf("Wrong match. expect: %d key lookups, got: %d", tt.expectCalls, calls.Load())
			}
		})
	}
}

func Test_hookMailgun_signatureWithoutSupplier(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &emailHandler{
		db:            dbhandler.NewMockDBHandler(mc),
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		timeNow:       testMailgunNow,
	}
	data := mailgunTestPayloadSigned("delivered", "", `{"voipbin_message_id":"`+testMailgunEmailID+`"}`, true, mailgunTestSignature())
	err := h.hookMailgun(context.Background(), data)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("Wrong match. expect: not configured, got: %v", err)
	}
}

func countingSecret(v string, calls *atomic.Int32) SecretFunc {
	return func(context.Context) (string, error) {
		calls.Add(1)
		return v, nil
	}
}
