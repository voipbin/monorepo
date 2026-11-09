package cachehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-call-manager/models/pushring"
)

func pushRingTestSession(now time.Time) *pushring.Session {
	return &pushring.Session{
		ID:           uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001"),
		CustomerID:   uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000002"),
		AgentID:      uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000003"),
		ExtensionID:  uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000004"),
		ConfbridgeID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000005"),
		MasterCallID: uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000006"),
		FlowID:       uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000007"),
		ExpiresAt:    now.Add(pushring.Window),
		State:        pushring.StateRinging,
		Tickets: map[string]uuid.UUID{
			"hash-a": uuid.FromStringOrNil("d0000000-0000-0000-0000-0000000000a1"),
			"hash-b": uuid.FromStringOrNil("d0000000-0000-0000-0000-0000000000b1"),
		},
	}
}

func Test_PushRingSessionCreateGet(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	now := time.Now().Truncate(time.Millisecond)
	s := pushRingTestSession(now)
	s.SIPRegistered = true

	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := h.PushRingSessionGet(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CustomerID != s.CustomerID || got.AgentID != s.AgentID || got.ExtensionID != s.ExtensionID ||
		got.ConfbridgeID != s.ConfbridgeID || got.MasterCallID != s.MasterCallID || got.FlowID != s.FlowID ||
		!got.ExpiresAt.Equal(s.ExpiresAt) || got.State != pushring.StateRinging || !got.SIPRegistered ||
		got.CallerJoined || got.Winner != "" || len(got.Tickets) != 2 {
		t.Errorf("wrong session. got: %+v", got)
	}

	for _, kind := range []struct {
		kind string
		id   uuid.UUID
	}{{pushring.IndexConfbridge, s.ConfbridgeID}, {pushring.IndexMaster, s.MasterCallID}, {pushring.IndexFlow, s.FlowID}} {
		sid, errGet := h.PushRingSessionIDGet(ctx, kind.kind, kind.id)
		if errGet != nil || sid != s.ID {
			t.Errorf("index %s: got %v err %v", kind.kind, sid, errGet)
		}
	}

	byMaster, byFlow, errIDs := h.PushRingSessionIDsGet(ctx, s.MasterCallID, s.FlowID)
	if errIDs != nil || byMaster != s.ID || byFlow != s.ID {
		t.Errorf("ids: got %v %v err %v", byMaster, byFlow, errIDs)
	}
	byMaster, byFlow, errIDs = h.PushRingSessionIDsGet(ctx, uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4()))
	if errIDs != nil || byMaster != uuid.Nil || byFlow != uuid.Nil {
		t.Errorf("ids of unknown keys must be nil. got %v %v err %v", byMaster, byFlow, errIDs)
	}

	if ttl := mr.TTL("call:pushring:" + s.ID.String()); ttl != pushring.KeyTTL {
		t.Errorf("session ttl: expected %v, got %v", pushring.KeyTTL, ttl)
	}
	if ttl := mr.TTL("call:pushticket:hash-a"); ttl != pushring.KeyTTL {
		t.Errorf("ticket ttl: expected %v, got %v", pushring.KeyTTL, ttl)
	}

	if _, err := h.PushRingSessionGet(ctx, uuid.Must(uuid.NewV4())); !errors.Is(err, pushring.ErrSessionNotFound) {
		t.Errorf("missing session: expected ErrSessionNotFound, got %v", err)
	}
	if _, err := h.PushRingSessionIDGet(ctx, pushring.IndexFlow, uuid.Must(uuid.NewV4())); !errors.Is(err, pushring.ErrSessionNotFound) {
		t.Errorf("missing index: expected ErrSessionNotFound, got %v", err)
	}
}

func Test_PushRingLeaseConfirm(t *testing.T) {
	ctx := context.Background()
	callA := uuid.FromStringOrNil("c0000000-0000-0000-0000-00000000000a")
	callB := uuid.FromStringOrNil("c0000000-0000-0000-0000-00000000000b")

	type step struct {
		name   string
		run    func(h *handler, s *pushring.Session, now time.Time) (bool, error)
		expect bool
	}

	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "one lease at a time, the second is refused",
			steps: []step{
				{"a takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"b refused", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now, now.Add(pushring.LeaseTTL))
				}, false},
				{"b cannot confirm", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callB, now, "sip")
				}, false},
				{"a confirms", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now, "sip")
				}, true},
				{"b refused after the winner", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now.Add(time.Minute-pushring.Window), now.Add(pushring.LeaseTTL))
				}, false},
			},
		},
		{
			name: "release lets the other call take",
			steps: []step{
				{"a takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"release by a", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return true, h.PushRingLeaseRelease(ctx, s.ID, callA)
				}, true},
				{"b takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"release by a does not free b", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					if err := h.PushRingLeaseRelease(ctx, s.ID, callA); err != nil {
						return false, err
					}
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, false},
			},
		},
		{
			name: "an expired lease can be taken and a late confirm of the old owner is refused",
			steps: []step{
				{"a takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"b takes after the lease expired", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now.Add(pushring.LeaseTTL+time.Second), now.Add(2*pushring.LeaseTTL))
				}, true},
				{"a late confirm refused", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now.Add(pushring.LeaseTTL+2*time.Second), "sip")
				}, false},
			},
		},
		{
			name: "a late confirm of the old owner is refused after another call became the winner",
			steps: []step{
				{"a takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"b takes after the lease of a expired", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now.Add(pushring.LeaseTTL+time.Second), now.Add(2*pushring.LeaseTTL))
				}, true},
				{"b confirms", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callB, now.Add(pushring.LeaseTTL+2*time.Second), "sip")
				}, true},
				{"a confirms after every lease expired", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now.Add(3*pushring.LeaseTTL), "sip")
				}, false},
			},
		},
		{
			name: "a late confirm re-takes an expired lease nobody else took",
			steps: []step{
				{"a takes", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"a confirms after its lease expired", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now.Add(pushring.LeaseTTL+time.Second), "sip")
				}, true},
			},
		},
		{
			name: "no lease and no confirm after the window",
			steps: []step{
				{"take after the window", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now.Add(pushring.Window), now.Add(pushring.Window+pushring.LeaseTTL))
				}, false},
				{"confirm after the window", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now.Add(pushring.Window), "sip")
				}, false},
			},
		},
		{
			name: "push_failed keeps the gate open, closed does not",
			steps: []step{
				{"push failed", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingFailed(ctx, s.ID)
				}, true},
				{"push failed again is a no-op", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingFailed(ctx, s.ID)
				}, false},
				{"a takes in push_failed", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callA, now, now.Add(pushring.LeaseTTL))
				}, true},
				{"close", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					got, err := h.PushRingClose(ctx, s.ID)
					return got != nil, err
				}, true},
				{"a cannot confirm after close", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingConfirm(ctx, s.ID, callA, now, "sip")
				}, false},
				{"b cannot take after close", func(h *handler, s *pushring.Session, now time.Time) (bool, error) {
					return h.PushRingLeaseTake(ctx, s.ID, callB, now, now.Add(pushring.LeaseTTL))
				}, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newCovHandler(t)
			now := time.Now().Truncate(time.Millisecond)
			s := pushRingTestSession(now)
			if err := h.PushRingSessionCreate(ctx, s); err != nil {
				t.Fatalf("create: %v", err)
			}

			for _, st := range tt.steps {
				got, err := st.run(h, s, now)
				if err != nil {
					t.Fatalf("%s: unexpected error: %v", st.name, err)
				}
				if got != st.expect {
					t.Errorf("%s: expected %v, got %v", st.name, st.expect, got)
				}
			}
		})
	}
}

func Test_PushRingCloseOnce(t *testing.T) {
	ctx := context.Background()
	h, _ := newCovHandler(t)
	now := time.Now()
	s := pushRingTestSession(now)
	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := h.PushRingClose(ctx, s.ID)
	if err != nil || got == nil {
		t.Fatalf("first close: got %v err %v", got, err)
	}
	if got.State != pushring.StateClosed || got.MasterCallID != s.MasterCallID || len(got.Tickets) != 2 {
		t.Errorf("closed session is wrong: %+v", got)
	}

	got, err = h.PushRingClose(ctx, s.ID)
	if err != nil || got != nil {
		t.Errorf("second close must report nothing. got %v err %v", got, err)
	}

	got, err = h.PushRingClose(ctx, uuid.Must(uuid.NewV4()))
	if err != nil || got != nil {
		t.Errorf("close of a missing session must report nothing. got %v err %v", got, err)
	}
}

func Test_PushRingTicket(t *testing.T) {
	ctx := context.Background()
	callA := uuid.FromStringOrNil("c0000000-0000-0000-0000-00000000000a")

	tests := []struct {
		name string

		prepare func(h *handler, s *pushring.Session)
		hash    string
		offset  time.Duration

		expectErr error
	}{
		{"ok", func(h *handler, s *pushring.Session) { _ = h.PushRingCallerJoined(ctx, s.ID) }, "hash-a", 0, nil},
		{"unknown ticket", func(h *handler, s *pushring.Session) { _ = h.PushRingCallerJoined(ctx, s.ID) }, "hash-x", 0, pushring.ErrTicketNotFound},
		{"before the caller joined", func(h *handler, s *pushring.Session) {}, "hash-a", 0, pushring.ErrTicketRefused},
		{"after the window", func(h *handler, s *pushring.Session) { _ = h.PushRingCallerJoined(ctx, s.ID) }, "hash-a", pushring.Window, pushring.ErrTicketRefused},
		{"session closed", func(h *handler, s *pushring.Session) {
			_ = h.PushRingCallerJoined(ctx, s.ID)
			_, _ = h.PushRingClose(ctx, s.ID)
		}, "hash-a", 0, pushring.ErrTicketRefused},
		{"after a winner", func(h *handler, s *pushring.Session) {
			_ = h.PushRingCallerJoined(ctx, s.ID)
			ok, _ := h.PushRingLeaseTake(ctx, s.ID, callA, time.Now(), time.Now().Add(pushring.LeaseTTL))
			if !ok {
				t.Fatal("lease")
			}
			if ok, _ := h.PushRingConfirm(ctx, s.ID, callA, time.Now(), "sip"); !ok {
				t.Fatal("confirm")
			}
		}, "hash-a", 0, pushring.ErrTicketRefused},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newCovHandler(t)
			now := time.Now().Truncate(time.Millisecond)
			s := pushRingTestSession(now)
			s.ExpiresAt = time.Now().Add(pushring.Window)
			if err := h.PushRingSessionCreate(ctx, s); err != nil {
				t.Fatalf("create: %v", err)
			}
			tt.prepare(h, s)

			at := time.Now().Add(tt.offset)
			got, err := h.PushRingTicketClaim(ctx, tt.hash, at, at.Add(pushring.TicketInUseTTL))
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("expected error %v, got %v", tt.expectErr, err)
			}
			if tt.expectErr == nil && (got == nil || got.SessionID != s.ID || got.DeviceID != s.Tickets[tt.hash] || got.TicketHash != tt.hash) {
				t.Errorf("wrong claim: %+v", got)
			}
		})
	}
}

func Test_PushRingTicketLifecycle(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	now := time.Now().Truncate(time.Millisecond)
	s := pushRingTestSession(now)
	s.ExpiresAt = now.Add(pushring.Window)
	callA := uuid.FromStringOrNil("c0000000-0000-0000-0000-00000000000a")
	callB := uuid.FromStringOrNil("c0000000-0000-0000-0000-00000000000b")
	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := h.PushRingCallerJoined(ctx, s.ID); err != nil {
		t.Fatalf("caller joined: %v", err)
	}

	// the first claim wins, a concurrent claim of the same ticket is refused
	if _, err := h.PushRingTicketClaim(ctx, "hash-a", now, now.Add(pushring.TicketInUseTTL)); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if _, err := h.PushRingTicketClaim(ctx, "hash-a", now.Add(time.Second), now.Add(time.Second+pushring.TicketInUseTTL)); !errors.Is(err, pushring.ErrTicketRefused) {
		t.Errorf("concurrent claim: expected refused, got %v", err)
	}

	// no call row was ever created: the ticket returns to issued by time (C4)
	later := now.Add(pushring.TicketInUseTTL + time.Second)
	if _, err := h.PushRingTicketClaim(ctx, "hash-a", later, later.Add(pushring.TicketInUseTTL)); err != nil {
		t.Errorf("claim after in_use_until: expected ok, got %v", err)
	}

	// the hangup hook of the ticket call returns the ticket to issued
	if err := h.PushRingTicketBind(ctx, s.ID, callA, "hash-a"); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if sid, err := h.PushRingSessionIDGet(ctx, pushring.IndexFlow, callA); err != nil || sid != s.ID {
		t.Errorf("flow index: got %v err %v", sid, err)
	}
	if got, errGet := h.PushRingSessionGet(ctx, s.ID); errGet != nil || got.TicketFlows[callA] != "hash-a" {
		t.Errorf("ticket flows: got %+v err %v", got, errGet)
	}
	if err := h.PushRingTicketReturn(ctx, s.ID, callB); err != nil { // another call: nothing changes
		t.Fatalf("return of another call: %v", err)
	}
	if got := mr.HGet("call:pushticket:hash-a", "state"); got != "in_use" {
		t.Errorf("another call must not return the ticket. state: %s", got)
	}
	if err := h.PushRingTicketReturn(ctx, s.ID, callA); err != nil {
		t.Fatalf("return: %v", err)
	}
	if got := mr.HGet("call:pushticket:hash-a", "state"); got != "issued" {
		t.Errorf("ticket must be issued again. state: %s", got)
	}

	// a consumed ticket never returns
	if _, err := h.PushRingTicketClaim(ctx, "hash-a", now, now.Add(pushring.TicketInUseTTL)); err != nil {
		t.Fatalf("claim again: %v", err)
	}
	if err := h.PushRingTicketBind(ctx, s.ID, callA, "hash-a"); err != nil {
		t.Fatalf("bind again: %v", err)
	}
	if err := h.PushRingTicketConsume(ctx, s.ID, callA); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if err := h.PushRingTicketReturn(ctx, s.ID, callA); err != nil {
		t.Fatalf("return after consume: %v", err)
	}
	if got := mr.HGet("call:pushticket:hash-a", "state"); got != "consumed" {
		t.Errorf("a consumed ticket must stay consumed. state: %s", got)
	}

	// a winner blocks the return
	if err := h.PushRingTicketBind(ctx, s.ID, callB, "hash-b"); err != nil {
		t.Fatalf("bind b: %v", err)
	}
	mr.HSet("call:pushticket:hash-b", "state", "in_use")
	if ok, _ := h.PushRingLeaseTake(ctx, s.ID, callB, now, now.Add(pushring.LeaseTTL)); !ok {
		t.Fatal("lease")
	}
	if ok, _ := h.PushRingConfirm(ctx, s.ID, callB, now, pushring.WinnerAppPrefix+"x"); !ok {
		t.Fatal("confirm")
	}
	if err := h.PushRingTicketReturn(ctx, s.ID, callB); err != nil {
		t.Fatalf("return after winner: %v", err)
	}
	if got := mr.HGet("call:pushticket:hash-b", "state"); got != "in_use" {
		t.Errorf("no ticket returns after a winner. state: %s", got)
	}

	// the tickets of the other devices are deleted
	got, err := h.PushRingSessionGet(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if err := h.PushRingTicketsDelete(ctx, got, s.Tickets["hash-b"]); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if mr.Exists("call:pushticket:hash-a") || !mr.Exists("call:pushticket:hash-b") {
		t.Errorf("only the ticket of the kept device must remain")
	}
	if err := h.PushRingTicketsDelete(ctx, got, uuid.Nil); err != nil {
		t.Fatalf("delete all: %v", err)
	}
	if mr.Exists("call:pushticket:hash-b") {
		t.Errorf("all tickets must be deleted")
	}
}

func Test_PushRingCacheFailureIsReported(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	s := pushRingTestSession(time.Now())
	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	mr.Close()

	if ok, err := h.PushRingLeaseTake(ctx, s.ID, uuid.Must(uuid.NewV4()), time.Now(), time.Now().Add(time.Second)); err == nil || ok {
		t.Errorf("a cache failure must be an error and never a granted lease. ok %v err %v", ok, err)
	}
	if _, err := h.PushRingSessionIDGet(ctx, pushring.IndexConfbridge, s.ConfbridgeID); err == nil || errors.Is(err, pushring.ErrSessionNotFound) {
		t.Errorf("a cache failure must not look like a missing key. err %v", err)
	}
}

// A ticket that was deleted while its call was being started is not created again by the bind or the consume. A key
// without a TTL would stay in Redis.
func Test_PushRingTicketNotRecreated(t *testing.T) {
	ctx := context.Background()
	h, mr := newCovHandler(t)
	now := time.Now().Truncate(time.Millisecond)
	s := pushRingTestSession(now)
	flowID := uuid.FromStringOrNil("c0000000-0000-0000-0000-0000000000f1")

	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := h.PushRingCallerJoined(ctx, s.ID); err != nil {
		t.Fatalf("caller joined: %v", err)
	}
	if _, err := h.PushRingTicketClaim(ctx, "hash-a", now, now.Add(pushring.TicketInUseTTL)); err != nil {
		t.Fatalf("claim: %v", err)
	}
	if err := h.PushRingTicketsDelete(ctx, s, uuid.Nil); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if err := h.PushRingTicketBind(ctx, s.ID, flowID, "hash-a"); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if mr.Exists("call:pushticket:hash-a") {
		t.Errorf("the bind must not create a deleted ticket again")
	}
	if mr.Exists("call:pushring:flow:" + flowID.String()) {
		t.Errorf("the bind of a deleted ticket must not index the flow")
	}

	// the consume of the flow that was never bound is a no-op as well
	if err := h.PushRingTicketConsume(ctx, s.ID, flowID); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if mr.Exists("call:pushticket:hash-a") {
		t.Errorf("the consume must not create a deleted ticket again")
	}

	// the session field of a bound flow whose ticket is deleted afterwards: consume must not recreate the key
	if err := h.PushRingSessionCreate(ctx, s); err != nil {
		t.Fatalf("create again: %v", err)
	}
	if err := h.PushRingTicketBind(ctx, s.ID, flowID, "hash-a"); err != nil {
		t.Fatalf("bind again: %v", err)
	}
	if err := h.PushRingTicketsDelete(ctx, s, uuid.Nil); err != nil {
		t.Fatalf("delete again: %v", err)
	}
	if err := h.PushRingTicketConsume(ctx, s.ID, flowID); err != nil {
		t.Fatalf("consume bound: %v", err)
	}
	if mr.Exists("call:pushticket:hash-a") {
		t.Errorf("the consume must not create a deleted ticket again")
	}
}
