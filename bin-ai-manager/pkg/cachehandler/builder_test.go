package cachehandler

import (
	"context"
	"strings"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
)

// The builder counter key is the contract between the daily limit and any
// operator who has to look the counter up or reset it. It is not a listen key,
// so it must not share that prefix.
func Test_builderChatCountKey(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	got := builderChatCountKey(customerID)
	if got != "ai:builder:chat:count:11111111-2222-3333-4444-555555555555" {
		t.Errorf("key mismatch. got: %s", got)
	}
	if strings.HasPrefix(got, "ai:listen:") {
		t.Errorf("the builder counter must not live under the listen prefix: %s", got)
	}
}

// The first INCR arms the TTL; later INCRs must not push the expiry forward,
// because the counter is a per-window daily limit and a customer who keeps
// calling would otherwise never see the window end.
func Test_BuilderChatCountIncr_armsTTLOnlyOnTheFirstIncrement(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	key := builderChatCountKey(customerID)

	got, err := h.BuilderChatCountIncr(ctx, customerID, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if got != 1 {
		t.Errorf("count mismatch. expected: 1, got: %d", got)
	}
	if ttl := mr.TTL(key); ttl != time.Hour {
		t.Errorf("the first increment must arm the TTL. expected: %s, got: %s", time.Hour, ttl)
	}

	// Let part of the window pass, then count again with a different TTL
	// argument: the remaining TTL must be untouched.
	mr.FastForward(10 * time.Minute)
	got, err = h.BuilderChatCountIncr(ctx, customerID, 24*time.Hour)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if got != 2 {
		t.Errorf("count mismatch. expected: 2, got: %d", got)
	}
	if ttl := mr.TTL(key); ttl != 50*time.Minute {
		t.Errorf("a later increment must not re-arm the TTL. expected: %s, got: %s", 50*time.Minute, ttl)
	}
}

// A counter that has lost its TTL is a key nothing reclaims, and it would lock
// the customer out for good. The next increment must heal it.
func Test_BuilderChatCountIncr_healsAKeyThatHasNoTTL(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	key := builderChatCountKey(customerID)
	if err := mr.Set(key, "7"); err != nil { // a key with no TTL at all
		t.Fatalf("setup failed. err: %v", err)
	}
	if ttl := mr.TTL(key); ttl != 0 {
		t.Fatalf("setup must leave the key without a TTL, got: %s", ttl)
	}

	got, err := h.BuilderChatCountIncr(context.Background(), customerID, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if got != 8 {
		t.Errorf("count mismatch. expected: 8, got: %d", got)
	}
	if ttl := mr.TTL(key); ttl != time.Hour {
		t.Errorf("a counter without a TTL must be given one. expected: %s, got: %s", time.Hour, ttl)
	}
}

// Counters are per customer.
func Test_BuilderChatCountIncr_isPerCustomer(t *testing.T) {
	a := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	b := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := h.BuilderChatCountIncr(ctx, a, time.Hour); err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
	}
	got, err := h.BuilderChatCountIncr(ctx, b, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if got != 1 {
		t.Errorf("another customer's count must start at 1, got: %d", got)
	}
}

// The key carries only the customer id, never any conversation text.
func Test_builderChatCountKey_hasNoContent(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	key := builderChatCountKey(customerID)

	rest := strings.TrimPrefix(key, "ai:builder:chat:count:")
	if _, err := uuid.FromString(rest); err != nil {
		t.Errorf("everything after the prefix must be the customer id only: %q", rest)
	}
}

// EXPIRE with 0 deletes the key. A zero or sub-second TTL (a misconfiguration)
// must therefore never turn a count into a delete, or the daily limit would
// silently reset on every call.
func Test_BuilderChatCountIncr_zeroTTLDoesNotDeleteTheCounter(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	for i := int64(1); i <= 3; i++ {
		got, err := h.BuilderChatCountIncr(ctx, customerID, 0)
		if err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
		if got != i {
			t.Fatalf("a zero TTL must not reset the count. expected: %d, got: %d", i, got)
		}
	}
}

// The Flow Builder counter has its own key, so the two builders never spend
// each other's daily allowance, and the Assistant key is unchanged.
func Test_builderFlowChatCountKey(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	got := builderFlowChatCountKey(customerID)
	if got != "ai:flow_builder:chat:count:11111111-2222-3333-4444-555555555555" {
		t.Errorf("key mismatch. got: %s", got)
	}
	if got == builderChatCountKey(customerID) {
		t.Errorf("the flow builder counter must not share the assistant builder key")
	}
}

func Test_BuilderFlowChatCountIncr_isIndependentOfTheAssistantCounter(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	h, mr := setupListenTestHandler(t)
	defer mr.Close()

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, err := h.BuilderChatCountIncr(ctx, customerID, time.Hour); err != nil {
			t.Fatalf("unexpected error. err: %v", err)
		}
	}

	got, err := h.BuilderFlowChatCountIncr(ctx, customerID, time.Hour)
	if err != nil {
		t.Fatalf("unexpected error. err: %v", err)
	}
	if got != 1 {
		t.Errorf("the flow counter must start at 1 regardless of the assistant counter. got: %d", got)
	}
	if ttl := mr.TTL(builderFlowChatCountKey(customerID)); ttl != time.Hour {
		t.Errorf("the first increment must arm the TTL. expected: %s, got: %s", time.Hour, ttl)
	}
}
