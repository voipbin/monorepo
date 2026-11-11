package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/internal/logtest"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

var errTestTokenRead = errors.New("the token store is down")

// A token that can not be read is a failed target, never a rejected one: a rejection makes the
// dispatch try the next provider, and a token error must not do that (the provider was not
// reached, and the replacement of a changed token by another path is not allowed).
func Test_classifyError_tokenUnavailable(t *testing.T) {
	err := fmt.Errorf("%w: telnyx: %w", requestexternal.ErrTokenUnavailable, errTestTokenRead)

	if got := classifyError(err); got != SendOutcomeFailed {
		t.Errorf("Wrong match. expect: %v, got: %v", SendOutcomeFailed, got)
	}
}

// The four providers on a request layer whose token can not be read: every target fails, and the
// request layer sends nothing (the supplier fails before a request is built).
func Test_providers_tokenUnavailable(t *testing.T) {
	failing := func(context.Context) (string, error) { return "", errTestTokenRead }
	ext := requestexternal.NewRequestExternal(failing, failing)

	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+1555****0001"}
	targets := testTargets("+1555****0002", "+1555****0003")
	urls := []string{"https://example.invalid/a.png"}
	ctx := context.Background()

	tests := []struct {
		name string
		send func() []SendResult

		expectLog string // a text that the log of the failed request holds
	}{
		{"telnyx sms", func() []SendResult { return NewSMSTelnyx(ext).SendSMS(ctx, uuid.Nil, source, targets, "hello", nil) }, "the provider token could not be read"},
		{"messagebird sms", func() []SendResult {
			return NewSMSMessagebird(ext).SendSMS(ctx, uuid.Nil, source, targets, "hello", nil)
		}, "the provider token could not be read"},
		{"telnyx mms", func() []SendResult {
			return NewMMSTelnyx(ext).SendMMS(ctx, uuid.Nil, source, targets, "hello", urls, nil)
		}, "The mms was not sent: the provider token could not be read"},
		{"messagebird mms", func() []SendResult {
			return NewMMSMessagebird(ext).SendMMS(ctx, uuid.Nil, source, targets, "hello", urls, nil)
		}, "The mms was not sent: the provider token could not be read"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := logtest.New(t)
			res := tt.send()
			if got := logs.Dump(); !strings.Contains(got, tt.expectLog) {
				t.Errorf("Wrong match. expect the log to hold: %s, got: %s", tt.expectLog, got)
			}
			if len(res) != len(targets) {
				t.Fatalf("Wrong match. expect: %d results, got: %d", len(targets), len(res))
			}
			for i, r := range res {
				if r.Outcome != SendOutcomeFailed {
					t.Errorf("Wrong match. target %d expect: %v, got: %v", i, SendOutcomeFailed, r.Outcome)
				}
			}
		})
	}
}
