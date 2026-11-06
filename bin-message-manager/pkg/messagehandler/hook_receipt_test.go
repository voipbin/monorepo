package messagehandler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
)

const (
	receiptRef  = "4031a1af-0000-4000-8000-0000000000aa"
	receiptHook = "https://hook.example.com/v1.0/hooks/telnyx"
)

// receiptPayload builds a telnyx receipt. recipients are phone number and status pairs.
func receiptPayload(eventType string, direction string, id string, parts int, recipients ...string) []byte {
	to := ""
	for i := 0; i+1 < len(recipients); i += 2 {
		if to != "" {
			to += ","
		}
		to += fmt.Sprintf(`{"phone_number":%q,"status":%q}`, recipients[i], recipients[i+1])
	}

	return []byte(fmt.Sprintf(`{"data":{"event_type":%q,"id":"evt-1","payload":{"direction":%q,"id":%q,"parts":%d,"from":{"phone_number":"+15550100001"},"to":[%s]}},"meta":{"attempt":1}}`,
		eventType, direction, id, parts, to))
}

type receiptFixture struct {
	db     *dbhandler.MockDBHandler
	notify *notifyhandler.MockNotifyHandler
	mem    *memDB
	ev     *events

	handler *messageHandler
	listed  int
}

func newReceiptFixture(t *testing.T, targets []target.Target) *receiptFixture {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	f := &receiptFixture{
		db:     dbhandler.NewMockDBHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
	}
	f.mem = newMemDB(f.db, storedMessage(targets))
	f.ev = expectEvents(f.notify)
	f.handler = &messageHandler{
		utilHandler:   newTestUtil(mc),
		db:            f.db,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: f.notify,
	}

	return f
}

// expectLookup answers the reference lookup with the stored message.
func (f *receiptFixture) expectLookup(ref string) {
	f.db.EXPECT().MessageListByProviderReference(gomock.Any(), ref, gomock.Any()).DoAndReturn(func(_ context.Context, _ string, _ time.Time) ([]*message.Message, error) {
		f.listed++
		f.mem.mu.Lock()
		defer f.mem.mu.Unlock()
		return []*message.Message{f.mem.snapshot()}, nil
	}).AnyTimes()
}

func receiptTargets() []target.Target {
	ts := withParts(testTargets(usDest1, usDest2), 1)
	ts[0].ProviderReferenceID = receiptRef
	ts[1].ProviderReferenceID = "4031a1af-0000-4000-8000-0000000000bb"
	return ts
}

func Test_Hook_receipts(t *testing.T) {
	tests := []struct {
		name string

		payload []byte

		storedStatus target.Status

		expectStatus  target.Status // of the matched target (index 0)
		expectParts   int
		expectEvents  int
		expectLookups int
	}{
		{"sent", receiptPayload("message.sent", "outbound", receiptRef, 2, usDest1, "sent"), target.StatusQueued, target.StatusSent, 2, 1, 1},
		{"delivered", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"), target.StatusQueued, target.StatusDelivered, 1, 1, 1},
		{"delivered after sent", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"), target.StatusSent, target.StatusDelivered, 1, 1, 1},
		{"delivery_failed", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivery_failed"), target.StatusQueued, target.StatusFailed, 1, 1, 1},
		{"sending_failed", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "sending_failed"), target.StatusQueued, target.StatusFailed, 1, 1, 1},
		{"delivery_unconfirmed", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivery_unconfirmed"), target.StatusQueued, target.StatusDLRTimeout, 1, 1, 1},
		{"a late sent never lowers delivered", receiptPayload("message.sent", "outbound", receiptRef, 1, usDest1, "sent"), target.StatusDelivered, target.StatusDelivered, 0, 0, 1},
		{"a duplicate delivered writes nothing", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"), target.StatusDelivered, target.StatusDelivered, 0, 0, 1},
		{"a final state is never replaced", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"), target.StatusFailed, target.StatusFailed, 0, 0, 1},
		{"an unknown status changes nothing", receiptPayload("message.sent", "outbound", receiptRef, 1, usDest1, "queued"), target.StatusQueued, target.StatusQueued, 0, 0, 1},
		{"the status is read from the entry of the target", receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest2, "delivery_failed", usDest1, "delivered"), target.StatusQueued, target.StatusDelivered, 1, 1, 1},
		{"no entry equals the destination", receiptPayload("message.finalized", "outbound", receiptRef, 1, "+19990000000", "delivered"), target.StatusQueued, target.StatusQueued, 0, 0, 1},
		{"a message that is not outbound is ignored", receiptPayload("message.finalized", "inbound", receiptRef, 1, usDest1, "delivered"), target.StatusQueued, target.StatusQueued, 0, 0, 0},
		{"an id that is not a uuid is ignored before any query", receiptPayload("message.finalized", "outbound", "not-a-uuid", 1, usDest1, "delivered"), target.StatusQueued, target.StatusQueued, 0, 0, 0},
		{"a receipt without a recipient is ignored", receiptPayload("message.finalized", "outbound", receiptRef, 1), target.StatusQueued, target.StatusQueued, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := receiptTargets()
			ts[0].Status = tt.storedStatus
			f := newReceiptFixture(t, ts)
			f.expectLookup(receiptRef)

			if err := f.handler.Hook(context.Background(), receiptHook, tt.payload); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			got := f.mem.targets()
			if got[0].Status != tt.expectStatus {
				t.Errorf("Wrong status. expect: %s, got: %s", tt.expectStatus, got[0].Status)
			}
			if got[0].ProviderParts != tt.expectParts {
				t.Errorf("Wrong provider parts. expect: %d, got: %d", tt.expectParts, got[0].ProviderParts)
			}
			if got[0].Parts != 1 {
				t.Errorf("The platform parts must not change. got: %d", got[0].Parts)
			}
			if got[1].Status != target.StatusQueued {
				t.Errorf("The other target must not change. got: %s", got[1].Status)
			}
			if f.ev.count() != tt.expectEvents {
				t.Errorf("Wrong event count. expect: %d, got: %d", tt.expectEvents, f.ev.count())
			}
			if f.listed != tt.expectLookups {
				t.Errorf("Wrong lookup count. expect: %d, got: %d", tt.expectLookups, f.listed)
			}
		})
	}
}

func Test_Hook_receipts_unmatched(t *testing.T) {
	f := newReceiptFixture(t, receiptTargets())
	before := promValue(t, "message_manager_receipt_unmatched_total", nil)
	otherRef := "4031a1af-0000-4000-8000-0000000000cc"

	// the lookup finds a row (the LIKE can be wider than an exact id), but no target has this exact id.
	f.expectLookup(otherRef)

	if err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", otherRef, 1, usDest1, "delivered")); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if got := promValue(t, "message_manager_receipt_unmatched_total", nil) - before; got != 1 {
		t.Errorf("Wrong unmatched count. expect: 1, got: %v", got)
	}
	if f.mem.writes() != 0 || f.ev.count() != 0 {
		t.Errorf("Nothing may change. writes: %d, events: %d", f.mem.writes(), f.ev.count())
	}
}

func Test_Hook_receipts_errors(t *testing.T) {
	t.Run("a lookup error asks for a retry", func(t *testing.T) {
		f := newReceiptFixture(t, receiptTargets())
		f.db.EXPECT().MessageListByProviderReference(gomock.Any(), receiptRef, gomock.Any()).Return(nil, errors.New("db down"))

		err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"))
		if !errors.Is(err, ErrReceiptRetry) {
			t.Errorf("Wrong match. expect: ErrReceiptRetry, got: %v", err)
		}
	})

	t.Run("a write error asks for a retry", func(t *testing.T) {
		f := newReceiptFixture(t, receiptTargets())
		f.expectLookup(receiptRef)
		f.mem.failIfErr = errors.New("db down")

		err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"))
		if !errors.Is(err, ErrReceiptRetry) {
			t.Errorf("Wrong match. expect: ErrReceiptRetry, got: %v", err)
		}
	})

	t.Run("conflicts that do not resolve ask for a retry", func(t *testing.T) {
		f := newReceiptFixture(t, receiptTargets())
		f.expectLookup(receiptRef)
		f.mem.forceConflicts = 1 << 30

		err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered"))
		if !errors.Is(err, ErrReceiptRetry) {
			t.Errorf("Wrong match. expect: ErrReceiptRetry, got: %v", err)
		}
		if f.mem.ifCalls != callbackWriteTries {
			t.Errorf("Wrong tries. expect: %d, got: %d", callbackWriteTries, f.mem.ifCalls)
		}
	})

	t.Run("a conflict that resolves is applied", func(t *testing.T) {
		f := newReceiptFixture(t, receiptTargets())
		f.expectLookup(receiptRef)
		f.mem.forceConflicts = 3

		if err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered")); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if got := f.mem.targets()[0].Status; got != target.StatusDelivered {
			t.Errorf("Wrong status. got: %s", got)
		}
	})

	t.Run("a failed read for the event does not fail the receipt", func(t *testing.T) {
		f := newReceiptFixture(t, receiptTargets())
		f.expectLookup(receiptRef)
		f.mem.failGetAfterWrite = true
		before := promValue(t, "message_manager_event_dropped_total", nil)

		// the write succeeds, then the read for the event fails: the event is dropped and the receipt is answered with
		// success, because a retried receipt would be a no-op.
		if err := f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", receiptRef, 1, usDest1, "delivered")); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
		if f.ev.count() != 0 {
			t.Errorf("Wrong event count. expect: 0, got: %d", f.ev.count())
		}
		if got := promValue(t, "message_manager_event_dropped_total", nil) - before; got != 1 {
			t.Errorf("Wrong dropped count. expect: 1, got: %v", got)
		}
	})
}

// Test_Hook_receipts_concurrent checks that receipts for several targets of one message that arrive together are all
// applied: every writer retries after a conflict, so no update is lost.
func Test_Hook_receipts_concurrent(t *testing.T) {
	const n = 20

	numbers := make([]string, n)
	for i := range numbers {
		numbers[i] = fmt.Sprintf("+1555010%04d", i)
	}
	ts := withParts(testTargets(numbers...), 1)
	for i := range ts {
		ts[i].ProviderReferenceID = fmt.Sprintf("4031a1af-0000-4000-8000-%012d", i)
	}

	f := newReceiptFixture(t, ts)
	f.db.EXPECT().MessageListByProviderReference(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, ref string, _ time.Time) ([]*message.Message, error) {
		f.mem.mu.Lock()
		defer f.mem.mu.Unlock()
		return []*message.Message{f.mem.snapshot()}, nil
	}).AnyTimes()

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- f.handler.Hook(context.Background(), receiptHook, receiptPayload("message.finalized", "outbound", ts[i].ProviderReferenceID, 1, numbers[i], "delivered"))
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("A receipt failed: %v", err)
		}
	}
	for i, got := range f.mem.targets() {
		if got.Status != target.StatusDelivered {
			t.Errorf("Receipt %d was lost. status: %s", i, got.Status)
		}
	}
	if f.ev.count() != n {
		t.Errorf("Wrong event count. expect: %d, got: %d", n, f.ev.count())
	}
}

func Test_updateTargets(t *testing.T) {
	ts := withParts(testTargets(usDest1), 1)

	t.Run("errNoChange writes nothing", func(t *testing.T) {
		f := newReceiptFixture(t, ts)
		changed, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: time.Second}, func([]target.Target) error { return errNoChange })
		if changed || err != nil || f.mem.writes() != 0 {
			t.Errorf("Wrong match. changed: %v, err: %v, writes: %d", changed, err, f.mem.writes())
		}
	})

	t.Run("a function error is returned without a write", func(t *testing.T) {
		f := newReceiptFixture(t, ts)
		want := errors.New("boom")
		_, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: time.Second}, func([]target.Target) error { return want })
		if !errors.Is(err, want) || f.mem.writes() != 0 {
			t.Errorf("Wrong match. err: %v, writes: %d", err, f.mem.writes())
		}
	})

	t.Run("the budget ends the tries", func(t *testing.T) {
		f := newReceiptFixture(t, ts)
		f.mem.forceConflicts = 1 << 30
		start := time.Now()
		_, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: 100 * time.Millisecond}, func([]target.Target) error { return nil })
		if !errors.Is(err, errUpdateConflict) {
			t.Errorf("Wrong match. expect: errUpdateConflict, got: %v", err)
		}
		if time.Since(start) > 2*time.Second {
			t.Errorf("The budget was not respected: %v", time.Since(start))
		}
	})

	t.Run("a read error is returned", func(t *testing.T) {
		f := newReceiptFixture(t, ts)
		f.mem.failGetFromDB = errors.New("db down")
		_, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: time.Second}, func([]target.Target) error { return nil })
		if err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})

	t.Run("the provider name is written only when given", func(t *testing.T) {
		f := newReceiptFixture(t, ts)
		name := message.ProviderNameMessagebird
		if _, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: time.Second}, func([]target.Target) error { return nil }); err != nil {
			t.Fatal(err)
		}
		if f.mem.providerName() != message.ProviderNameTelnyx {
			t.Errorf("The provider name changed. got: %s", f.mem.providerName())
		}
		if _, err := f.handler.updateTargets(context.Background(), testMessageID, updateOpts{budget: time.Second, provider: &name}, func([]target.Target) error { return nil }); err != nil {
			t.Fatal(err)
		}
		if f.mem.providerName() != message.ProviderNameMessagebird {
			t.Errorf("The provider name was not written. got: %s", f.mem.providerName())
		}
	})
}
