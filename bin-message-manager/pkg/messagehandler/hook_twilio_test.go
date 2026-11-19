package messagehandler

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // the signature of the Twilio callbacks is HMAC-SHA1 by the definition of Twilio.
	"encoding/base64"
	"errors"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

const (
	twilioTestSID      = "SM0123456789abcdef0123456789abcdef"
	twilioTestOtherSID = "SM1123456789abcdef0123456789abcdef"
	twilioTestToken    = "fake-twilio-token"
	twilioTestCallback = "https://hook.example.com/v1.0/messages/twilio"
	twilioTestHook     = "hook.example.com/v1.0/messages/twilio"
)

// twilioTestSignature signs a form the way Twilio does: the callback URL, then every key and value in the order of the
// keys.
func twilioTestSignature(token string, callback string, form url.Values) string {
	keys := make([]string, 0, len(form))
	for k := range form {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString(callback)
	for _, k := range keys {
		for _, v := range form[k] {
			sb.WriteString(k + v)
		}
	}

	mac := hmac.New(sha1.New, []byte(token))
	mac.Write([]byte(sb.String()))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func twilioTestConfig(token string, callback string) requestexternal.TwilioConfig {
	return requestexternal.TwilioConfig{
		Credentials: func(context.Context) (string, string, error) { return "AC-fake", token, nil },
		CallbackURL: func(context.Context) string { return callback },
	}
}

func twilioTestTargets() []target.Target {
	ts := withParts(testTargets(usDest1, usDest2), 1)
	ts[0].ProviderReferenceID = twilioTestSID
	ts[1].ProviderReferenceID = twilioTestOtherSID

	return ts
}

func newTwilioHookFixture(t *testing.T, config requestexternal.TwilioConfig, targets []target.Target) *receiptFixture {
	t.Helper()

	f := newReceiptFixture(t, targets)
	f.handler.reqExternal = requestexternal.NewRequestExternal(nil, nil, config)

	return f
}

func rejectedTwilio(t *testing.T, reason string) float64 {
	t.Helper()

	return promValue(t, "message_manager_twilio_callback_rejected_total", map[string]string{"reason": reason})
}

func Test_Hook_twilioStatus(t *testing.T) {
	tests := []struct {
		name string

		form   url.Values
		stored target.Status

		expectStatus target.Status
		expectEvents int
		expectLookup bool
	}{
		{"sent", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"sent"}}, target.StatusQueued, target.StatusSent, 1, true},
		{"delivered", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}, "To": {usDest1}}, target.StatusQueued, target.StatusDelivered, 1, true},
		{"delivered after sent", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}}, target.StatusSent, target.StatusDelivered, 1, true},
		{"failed", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"failed"}, "ErrorCode": {"30003"}}, target.StatusQueued, target.StatusFailed, 1, true},
		{"undelivered", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"undelivered"}}, target.StatusSent, target.StatusFailed, 1, true},
		{"canceled", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"canceled"}}, target.StatusQueued, target.StatusFailed, 1, true},
		{"a late sent never lowers delivered", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"sent"}}, target.StatusDelivered, target.StatusDelivered, 0, true},
		{"a duplicate delivered writes nothing", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}}, target.StatusDelivered, target.StatusDelivered, 0, true},
		{"a final state is never replaced", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}}, target.StatusFailed, target.StatusFailed, 0, true},
		{"queued changes nothing and asks nothing", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"queued"}}, target.StatusQueued, target.StatusQueued, 0, false},
		{"sending changes nothing", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"sending"}}, target.StatusQueued, target.StatusQueued, 0, false},
		{"an unknown status changes nothing", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"something_new"}}, target.StatusQueued, target.StatusQueued, 0, false},
		{"a new parameter is signed and ignored", url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}, "NewField": {"x"}}, target.StatusQueued, target.StatusDelivered, 1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := twilioTestTargets()
			ts[0].Status = tt.stored
			f := newTwilioHookFixture(t, twilioTestConfig(twilioTestToken, twilioTestCallback), ts)
			f.expectLookup(twilioTestSID)

			body := tt.form.Encode()
			sig := twilioTestSignature(twilioTestToken, twilioTestCallback, tt.form)
			if err := f.handler.Hook(context.Background(), twilioTestHook, []byte(body), sig); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got := f.mem.targets()
			if got[0].Status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %s, got: %s", tt.expectStatus, got[0].Status)
			}
			if got[1].Status != target.StatusQueued {
				t.Errorf("Another target was changed: %s", got[1].Status)
			}
			if f.ev.count() != tt.expectEvents {
				t.Errorf("Wrong event count. expect: %d, got: %d", tt.expectEvents, f.ev.count())
			}
			if (f.listed > 0) != tt.expectLookup {
				t.Errorf("Wrong lookup. expect: %v, got: %d", tt.expectLookup, f.listed)
			}
		})
	}
}

// A callback that is refused changes nothing, asks nothing, and is counted by its reason. The rejection is never a
// receipt retry.
func Test_Hook_twilioRefused(t *testing.T) {
	form := url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"delivered"}}
	body := []byte(form.Encode())
	good := twilioTestSignature(twilioTestToken, twilioTestCallback, form)

	tests := []struct {
		name string

		config    requestexternal.TwilioConfig
		body      []byte
		signature string

		expectReason string
	}{
		{"no signature", twilioTestConfig(twilioTestToken, twilioTestCallback), body, "", "missing_signature"},
		{"wrong signature", twilioTestConfig(twilioTestToken, twilioTestCallback), body, "AAAA" + good[4:], "invalid_signature"},
		{"a body that was changed", twilioTestConfig(twilioTestToken, twilioTestCallback), []byte(url.Values{"MessageSid": {twilioTestSID}, "MessageStatus": {"failed"}}.Encode()), good, "invalid_signature"},
		{"signed for another url", twilioTestConfig(twilioTestToken, "https://hook.example.com/other/messages/twilio"), body, good, "invalid_signature"},
		{"no auth token", twilioTestConfig("", twilioTestCallback), body, good, "not_configured"},
		{"no callback url", twilioTestConfig(twilioTestToken, ""), body, good, "not_configured"},
		{"no twilio configuration at all", requestexternal.TwilioConfig{}, body, good, "not_configured"},
		{"a form that cannot be parsed", twilioTestConfig(twilioTestToken, twilioTestCallback), []byte("MessageStatus=%zz"), good, "malformed"},
		{"a body that is too large", twilioTestConfig(twilioTestToken, twilioTestCallback), []byte("A=" + strings.Repeat("a", twilioCallbackMaxBody)), good, "too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTwilioHookFixture(t, tt.config, twilioTestTargets())
			// no lookup is expected: the mock fails the test on any database call.

			before := rejectedTwilio(t, tt.expectReason)
			err := f.handler.Hook(context.Background(), twilioTestHook, tt.body, tt.signature)

			if !errors.Is(err, ErrHookUnauthorized) {
				t.Fatalf("Wrong match. expect: ErrHookUnauthorized, got: %v", err)
			}
			if errors.Is(err, ErrReceiptRetry) {
				t.Errorf("A refused callback must never be a receipt retry: %v", err)
			}
			if strings.Contains(err.Error(), twilioTestSID) || strings.Contains(err.Error(), twilioTestToken) || strings.Contains(err.Error(), usDest1) {
				t.Errorf("The error holds a value of the request: %v", err)
			}
			if got := rejectedTwilio(t, tt.expectReason) - before; got != 1 {
				t.Errorf("Wrong rejected count of %s. expect: 1, got: %v", tt.expectReason, got)
			}
			if got := f.mem.targets(); got[0].Status != target.StatusQueued || f.ev.count() != 0 {
				t.Errorf("A refused callback changed a target. got: %v", got)
			}
		})
	}
}

// A signed body that is not a status callback creates nothing and changes nothing. The mock database has no create call.
func Test_Hook_twilioNotAStatusCallback(t *testing.T) {
	tests := []struct {
		name string
		form url.Values
	}{
		{"an incoming message", url.Values{"MessageSid": {twilioTestSID}, "SmsStatus": {"received"}, "From": {usDest2}, "To": {usDest1}, "Body": {"hi"}, "NumMedia": {"0"}}},
		{"an empty body", url.Values{}},
		{"a status without an id", url.Values{"MessageStatus": {"delivered"}}},
		{"an id that is not a twilio id", url.Values{"MessageSid": {"4031a1af-0000-4000-8000-0000000000aa"}, "MessageStatus": {"delivered"}}},
		{"an id with the wrong prefix", url.Values{"MessageSid": {"XX0123456789abcdef0123456789abcdef"}, "MessageStatus": {"delivered"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newTwilioHookFixture(t, twilioTestConfig(twilioTestToken, twilioTestCallback), twilioTestTargets())

			sig := twilioTestSignature(twilioTestToken, twilioTestCallback, tt.form)
			if err := f.handler.Hook(context.Background(), twilioTestHook, []byte(tt.form.Encode()), sig); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if f.ev.count() != 0 || f.mem.writes() != 0 {
				t.Errorf("Wrong match. expect: nothing written, got events: %d, writes: %d", f.ev.count(), f.mem.writes())
			}
		})
	}
}

// An unknown message id is counted and answered with success; a database error is a retry, like the telnyx receipt.
func Test_Hook_twilioLookup(t *testing.T) {
	form := url.Values{"MessageSid": {"SMffffffffffffffffffffffffffffffff"}, "MessageStatus": {"delivered"}}
	sig := twilioTestSignature(twilioTestToken, twilioTestCallback, form)

	t.Run("unknown id", func(t *testing.T) {
		f := newTwilioHookFixture(t, twilioTestConfig(twilioTestToken, twilioTestCallback), twilioTestTargets())
		f.db.EXPECT().MessageListByProviderReference(gomock.Any(), "SMffffffffffffffffffffffffffffffff", gomock.Any()).Return(nil, nil)

		before := promValue(t, "message_manager_receipt_unmatched_total", nil)
		if err := f.handler.Hook(context.Background(), twilioTestHook, []byte(form.Encode()), sig); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if got := promValue(t, "message_manager_receipt_unmatched_total", nil) - before; got != 1 {
			t.Errorf("Wrong unmatched count. expect: 1, got: %v", got)
		}
	})

	t.Run("database error", func(t *testing.T) {
		f := newTwilioHookFixture(t, twilioTestConfig(twilioTestToken, twilioTestCallback), twilioTestTargets())
		f.db.EXPECT().MessageListByProviderReference(gomock.Any(), "SMffffffffffffffffffffffffffffffff", gomock.Any()).DoAndReturn(
			func(context.Context, string, time.Time) ([]*message.Message, error) {
				return nil, errors.New("db down")
			})

		err := f.handler.Hook(context.Background(), twilioTestHook, []byte(form.Encode()), sig)
		if !errors.Is(err, ErrReceiptRetry) || errors.Is(err, ErrHookUnauthorized) {
			t.Errorf("Wrong match. expect: a receipt retry that is not a refusal, got: %v", err)
		}
	})
}

// The telnyx hook and an unknown hook are not changed by the signature argument.
func Test_Hook_signatureIsOnlyForTwilio(t *testing.T) {
	f := newTwilioHookFixture(t, twilioTestConfig(twilioTestToken, twilioTestCallback), twilioTestTargets())
	f.expectLookup(receiptRef)

	// a telnyx receipt for an id that no target holds: handled as before, with or without a signature.
	for _, sig := range []string{"", "some-signature"} {
		if err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"), sig); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	}
	if err := f.handler.Hook(context.Background(), "hook.example.com/v1.0/messages/other", []byte("{}"), ""); err == nil || errors.Is(err, ErrHookUnauthorized) {
		t.Errorf("Wrong match. expect: the unknown hook error, got: %v", err)
	}
}

// The mapping table of the twilio statuses.
func Test_mapTwilioStatus(t *testing.T) {
	tests := map[string]target.Status{
		"sent": target.StatusSent, "delivered": target.StatusDelivered,
		"failed": target.StatusFailed, "undelivered": target.StatusFailed, "canceled": target.StatusFailed,
		"queued": "", "accepted": "", "scheduled": "", "sending": "", "receiving": "", "received": "", "read": "", "partially_delivered": "", "": "", "other": "",
	}
	for in, expect := range tests {
		if got := mapTwilioStatus(in); got != expect {
			t.Errorf("Wrong match. status: %q, expect: %q, got: %q", in, expect, got)
		}
	}
}
