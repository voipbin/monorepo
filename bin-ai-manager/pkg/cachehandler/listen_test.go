package cachehandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	uuid "github.com/gofrs/uuid"
)

// Test_listenKeys pins every listen Redis key format. These keys are the
// contract between the intake path (which resolves ownership by SMEMBERS on a
// key built from a transcribe id) and the listen lifecycle (which writes and
// removes membership on the same key). A format drift between the two would
// silently stop every transcript segment from ever being matched -- with no
// error and no metric, since "not one of ours" is the overwhelmingly common
// case the intake path is designed to drop cheaply.
func Test_listenKeys(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	tests := []struct {
		name   string
		got    string
		expect string
	}{
		{"transcribe resolver set", listenTranscribeKey(transcribeID), "ai:listen:transcribe:11111111-2222-3333-4444-555555555555"},
		{"pending buffer list", listenPendingKey(aicallID), "ai:listen:pending:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"rolling window list", listenWindowKey(aicallID), "ai:listen:window:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"debounce lock", listenLockKey(aicallID), "ai:listen:lock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn counter", listenTurnsKey(aicallID), "ai:listen:turns:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"turn pipecatcall id set", listenTurnPipecatcallIDKey(aicallID), "ai:listen:turnpcid:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		// The start lock's key format is built in exactly ONE place, inside
		// ListenStartLockAcquire/Release (design §5.2.2, review round 16
		// finding LOW-6). Pinning it here is what keeps a future caller from
		// re-deriving it inline and drifting.
		{"start lock", listenStartLockKey(aicallID), "ai:listen:startlock:66666666-7777-8888-9999-aaaaaaaaaaaa"},
		{"conversation resolver set", listenConversationKey(transcribeID), "ai:listen:conversation:11111111-2222-3333-4444-555555555555"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expect {
				t.Errorf("key mismatch.\nexpected: %s\ngot:      %s", tt.expect, tt.got)
			}
		})
	}
}

// Test_listenPendingPopMax pins the atomic-drain bound. LPOP key count (Redis
// >= 6.2) is what makes draining the pending buffer a single atomic command, so
// a concurrent appender can never lose a line between a read and a trim. The
// count argument is REQUIRED by go-redis v8's LPopCount, so there must be a
// bound; it exists to cap one turn's context, not to be tuned.
func Test_listenPendingPopMax(t *testing.T) {
	if listenPendingPopMax != 500 {
		t.Errorf("listenPendingPopMax mismatch. expected: 500, got: %d", listenPendingPopMax)
	}
}

// setupListenTestHandler wires a handler against an in-process Redis, so the
// EVAL scripts below are actually EXECUTED rather than merely string-compared.
func setupListenTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis. err: %v", err)
	}

	return &handler{
		Addr:  mr.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}, mr
}

// Test_listenWritesAreAtomicWithTheirTTL is the direct regression test for
// review round 1's MEDIUM-4.
//
// Each of these was a write followed by a SEPARATE Expire. A failed Expire left
// the key with NO TTL at all -- for the resolver set specifically, that defeats
// listenResolverTTL outright and lets a lost cleanup strand this AIcall's
// subscription to a transcribe session forever. Each is now one EVAL, so the
// write cannot land without its TTL.
func Test_listenWritesAreAtomicWithTheirTTL(t *testing.T) {
	transcribeID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")
	pipecatcallID := uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")

	ttl := 30 * time.Minute

	tests := []struct {
		name string

		write     func(h *handler) error
		key       string
		expectTTL time.Duration
	}{
		{
			name: "ListenAIcallIDAdd",
			write: func(h *handler) error {
				return h.ListenAIcallIDAdd(context.Background(), transcribeID, aicallID, ttl)
			},
			key:       listenTranscribeKey(transcribeID),
			expectTTL: ttl,
		},
		{
			name: "ListenPendingPush",
			write: func(h *handler) error {
				return h.ListenPendingPush(context.Background(), aicallID, "[CUSTOMER] hello", ttl)
			},
			key:       listenPendingKey(aicallID),
			expectTTL: ttl,
		},
		{
			name: "ListenWindowPush",
			write: func(h *handler) error {
				return h.ListenWindowPush(context.Background(), aicallID, "[CUSTOMER] hello", 40, ttl)
			},
			key:       listenWindowKey(aicallID),
			expectTTL: ttl,
		},
		{
			name: "ListenTurnPipecatcallIDAdd",
			write: func(h *handler) error {
				return h.ListenTurnPipecatcallIDAdd(context.Background(), aicallID, pipecatcallID, ttl)
			},
			key:       listenTurnPipecatcallIDKey(aicallID),
			expectTTL: ttl,
		},
		{
			// Review round 2 MEDIUM-3: this was the last write in the file
			// still on the INCR-then-separate-EXPIRE idiom the four above were
			// converted away from.
			name: "ListenTurnCountIncr",
			write: func(h *handler) error {
				_, err := h.ListenTurnCountIncr(context.Background(), aicallID, ttl)
				return err
			},
			key:       listenTurnsKey(aicallID),
			expectTTL: ttl,
		},
		{
			// VOIP-1470: the conversation resolver reuses listenSetAddExpireScript,
			// so its add must carry its TTL atomically like the transcribe set.
			name: "ListenConversationAIcallIDAdd",
			write: func(h *handler) error {
				return h.ListenConversationAIcallIDAdd(context.Background(), transcribeID, aicallID, ttl)
			},
			key:       listenConversationKey(transcribeID),
			expectTTL: ttl,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := setupListenTestHandler(t)
			defer mr.Close()

			if err := tt.write(h); err != nil {
				t.Fatalf("unexpected error. err: %v", err)
			}

			if !mr.Exists(tt.key) {
				t.Fatalf("the write did not land. key: %s", tt.key)
			}
			if got := mr.TTL(tt.key); got != tt.expectTTL {
				t.Errorf("the TTL must be armed by the SAME command that wrote. key: %s, expected: %s, got: %s", tt.key, tt.expectTTL, got)
			}
		})
	}
}

// Test_ListenWindowPush_TrimsInsideTheSameScript pins that the window's bound
// is enforced by the same atomic write, not by a separate follow-up command.
func Test_ListenWindowPush_TrimsInsideTheSameScript(t *testing.T) {
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if err := h.ListenWindowPush(ctx, aicallID, fmt.Sprintf("line-%d", i), 3, time.Hour); err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
	}

	got, err := h.ListenWindowGet(ctx, aicallID)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}

	expect := []string{"line-2", "line-3", "line-4"}
	if len(got) != len(expect) {
		t.Fatalf("window size mismatch. expected: %v, got: %v", expect, got)
	}
	for i := range expect {
		if got[i] != expect[i] {
			t.Errorf("window content mismatch at %d. expected: %s, got: %s", i, expect[i], got[i])
		}
	}
}

// Test_ListenTurnCountIncr_ReturnsTheIncrementedCount pins that moving this
// function onto a Lua script did not change its contract: it must still return
// the counter's NEW value on every call, since that value is what the hard turn
// cap compares against.
func Test_ListenTurnCountIncr_ReturnsTheIncrementedCount(t *testing.T) {
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	for i := int64(1); i <= 3; i++ {
		got, err := h.ListenTurnCountIncr(ctx, aicallID, time.Hour)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if got != i {
			t.Errorf("the incremented count must be returned. expected: %d, got: %d", i, got)
		}
	}

	// And the TTL is re-armed by every one of those calls, not only the first.
	if got := mr.TTL(listenTurnsKey(aicallID)); got != time.Hour {
		t.Errorf("the TTL must be armed by the SAME command that incremented. expected: %s, got: %s", time.Hour, got)
	}
}

// Test_listenTTLSeconds pins the floor at 1. EXPIRE with 0 DELETES the key, so
// a sub-second TTL must never be allowed to turn a write into a delete.
func Test_listenTTLSeconds(t *testing.T) {
	tests := []struct {
		name   string
		ttl    time.Duration
		expect int
	}{
		{"a whole-second ttl passes through", 30 * time.Second, 30},
		{"an hour is rendered in seconds", time.Hour, 3600},
		{"a sub-second ttl floors at 1, never 0", 500 * time.Millisecond, 1},
		{"a zero ttl floors at 1", 0, 1},
		{"a negative ttl floors at 1", -5 * time.Second, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenTTLSeconds(tt.ttl); got != tt.expect {
				t.Errorf("mismatch. expected: %d, got: %d", tt.expect, got)
			}
		})
	}
}

// Test_listenConversationResolverAndPendingLen exercises the VOIP-1470
// primitives end to end: membership add/is-member/get/remove and the pending
// list length the conversation turn short-circuits on.
func Test_listenConversationResolverAndPendingLen(t *testing.T) {
	conversationID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")
	otherID := uuid.FromStringOrNil("77777777-7777-8888-9999-aaaaaaaaaaaa")
	ctx := context.Background()

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	if member, err := h.ListenConversationAIcallIDIsMember(ctx, conversationID, aicallID); err != nil || member {
		t.Fatalf("fresh key must not have members. member: %v, err: %v", member, err)
	}
	if err := h.ListenConversationAIcallIDAdd(ctx, conversationID, aicallID, time.Hour); err != nil {
		t.Fatalf("add failed. err: %v", err)
	}
	if err := h.ListenConversationAIcallIDAdd(ctx, conversationID, otherID, time.Hour); err != nil {
		t.Fatalf("second add failed. err: %v", err)
	}
	if member, err := h.ListenConversationAIcallIDIsMember(ctx, conversationID, aicallID); err != nil || !member {
		t.Fatalf("added id must be a member. member: %v, err: %v", member, err)
	}
	got, err := h.ListenConversationAIcallIDsGet(ctx, conversationID)
	if err != nil || len(got) != 2 {
		t.Fatalf("get must return both members. got: %v, err: %v", got, err)
	}
	if err := h.ListenConversationAIcallIDRemove(ctx, conversationID, aicallID); err != nil {
		t.Fatalf("remove failed. err: %v", err)
	}
	got, err = h.ListenConversationAIcallIDsGet(ctx, conversationID)
	if err != nil || len(got) != 1 || got[0] != otherID {
		t.Fatalf("remove must leave only the other member. got: %v, err: %v", got, err)
	}

	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 0 {
		t.Fatalf("empty pending list must be 0. n: %d, err: %v", n, err)
	}
	for i := 0; i < 3; i++ {
		if err := h.ListenPendingPush(ctx, aicallID, fmt.Sprintf("[CUSTOMER] %d", i), time.Hour); err != nil {
			t.Fatalf("push failed. err: %v", err)
		}
	}
	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 3 {
		t.Fatalf("pending list must count pushes. n: %d, err: %v", n, err)
	}
	if _, err := h.ListenPendingPopAll(ctx, aicallID); err != nil {
		t.Fatalf("pop failed. err: %v", err)
	}
	if n, err := h.ListenPendingLen(ctx, aicallID); err != nil || n != 0 {
		t.Fatalf("drained pending list must be 0. n: %d, err: %v", n, err)
	}
}
