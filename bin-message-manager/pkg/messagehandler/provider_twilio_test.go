package messagehandler

import (
	"context"
	"errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/models/twilio"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// The twilio providers classify the answer of each target like the other providers: accepted with the id and the
// segment count, a definitive refusal, and everything else as not known.
func Test_twilioProviders(t *testing.T) {
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: usSource}
	targets := []target.Target{testTargets(usDest1)[0], testTargets(usDest2)[0], testTargets(krDest)[0], testTargets("+1555****0009")[0]}
	urls := []string{"https://example.invalid/a.png"}

	answers := []struct {
		res *twilio.MessageResponse
		err error
	}{
		{&twilio.MessageResponse{SID: "SM1", NumSegments: "3"}, nil},
		{nil, &requestexternal.ProviderError{Provider: "twilio", StatusCode: 400}},
		{nil, &requestexternal.ProviderError{Provider: "twilio", StatusCode: 503}},
		{nil, errors.New("transport")},
	}
	expect := []SendResult{
		{Outcome: SendOutcomeAccepted, ProviderReferenceID: "SM1", ProviderParts: 3},
		{Outcome: SendOutcomeRejected},
		{Outcome: SendOutcomeFailed},
		{Outcome: SendOutcomeFailed},
	}

	check := func(t *testing.T, got []SendResult) {
		t.Helper()
		if len(got) != len(expect) {
			t.Fatalf("Wrong match. expect: %v, got: %v", expect, got)
		}
		for i := range expect {
			if got[i] != expect[i] {
				t.Errorf("Wrong result %d. expect: %v, got: %v", i, expect[i], got[i])
			}
		}
	}

	t.Run("sms", func(t *testing.T) {
		mockExternal := requestexternal.NewMockRequestExternal(gomock.NewController(t))
		h := NewSMSTwilio(mockExternal)
		before := promValue(t, "message_manager_twilio_number_send_total", map[string]string{"type": "sms"})

		for i, tg := range targets {
			mockExternal.EXPECT().TwilioSendSMS(gomock.Any(), usSource, tg.Destination.Target, "hello").Return(answers[i].res, answers[i].err)
		}

		check(t, h.SendSMS(context.Background(), uuid.Nil, source, targets, "hello", nil))

		if got := promValue(t, "message_manager_twilio_number_send_total", map[string]string{"type": "sms"}) - before; got != 1 {
			t.Errorf("Wrong send count. expect: 1, got: %v", got)
		}
	})

	t.Run("mms", func(t *testing.T) {
		mockExternal := requestexternal.NewMockRequestExternal(gomock.NewController(t))
		h := NewMMSTwilio(mockExternal)
		before := promValue(t, "message_manager_twilio_number_send_total", map[string]string{"type": "mms"})

		for i, tg := range targets {
			mockExternal.EXPECT().TwilioSendMMS(gomock.Any(), usSource, tg.Destination.Target, "hello", urls).Return(answers[i].res, answers[i].err)
		}

		check(t, h.SendMMS(context.Background(), uuid.Nil, source, targets, "hello", urls, nil))

		if got := promValue(t, "message_manager_twilio_number_send_total", map[string]string{"type": "mms"}) - before; got != 1 {
			t.Errorf("Wrong send count. expect: 1, got: %v", got)
		}
	})

	t.Run("an mms that could not be sent because the credentials are gone is not a rejection", func(t *testing.T) {
		mockExternal := requestexternal.NewMockRequestExternal(gomock.NewController(t))
		h := NewMMSTwilio(mockExternal)
		mockExternal.EXPECT().TwilioSendMMS(gomock.Any(), usSource, usDest1, "x", urls).Return(nil, requestexternal.ErrTokenUnavailable)

		got := h.SendMMS(context.Background(), uuid.Nil, source, targets[:1], "x", urls, nil)
		if len(got) != 1 || got[0].Outcome != SendOutcomeFailed {
			t.Errorf("Wrong match. expect: failed, got: %v", got)
		}
	})
}
