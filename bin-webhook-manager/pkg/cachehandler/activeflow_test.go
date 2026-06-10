package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"
)

func setupTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}

	cache := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	h := &handler{
		Addr:  mr.Addr(),
		Cache: cache,
	}

	return h, mr
}

// Test_ActiveflowWebhook_PositiveNegativeRoundTrip verifies a positive and a
// negative entry survive a set/get round trip.
func Test_ActiveflowWebhook_PositiveNegativeRoundTrip(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	// positive round trip.
	idPos := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	tmPos := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	wPos := &mwactiveflow.Webhook{
		URI:    "af.test.com",
		Method: webhook.MethodTypePOST,
		Tm:     tmPos,
	}
	if err := h.ActiveflowWebhookSet(ctx, idPos, wPos, time.Hour); err != nil {
		t.Fatalf("Could not set the positive entry. err: %v", err)
	}

	gotPos, found, err := h.ActiveflowWebhookGet(ctx, idPos)
	if err != nil {
		t.Fatalf("Could not get the positive entry. err: %v", err)
	}
	if !found {
		t.Fatalf("Wrong match. expect: found, got: miss")
	}
	if !gotPos.IsPositive() || gotPos.URI != "af.test.com" || gotPos.Method != webhook.MethodTypePOST {
		t.Errorf("Wrong match. expect positive af.test.com, got: %v", gotPos)
	}
	if !gotPos.Tm.Equal(tmPos) {
		t.Errorf("Wrong match. expect tm: %v, got: %v", tmPos, gotPos.Tm)
	}

	// negative round trip.
	idNeg := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	tmNeg := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)
	tmDelete := time.Date(2026, 6, 10, 1, 0, 0, 0, time.UTC)
	if err := h.ActiveflowWebhookSetNegative(ctx, idNeg, tmNeg, &tmDelete, time.Hour); err != nil {
		t.Fatalf("Could not set the negative entry. err: %v", err)
	}

	gotNeg, found, err := h.ActiveflowWebhookGet(ctx, idNeg)
	if err != nil {
		t.Fatalf("Could not get the negative entry. err: %v", err)
	}
	if !found {
		t.Fatalf("Wrong match. expect: found, got: miss")
	}
	if gotNeg.IsPositive() {
		t.Errorf("Wrong match. expect negative, got positive: %v", gotNeg)
	}
	if gotNeg.TMDelete == nil || !gotNeg.TMDelete.Equal(tmDelete) {
		t.Errorf("Wrong match. expect tm_delete: %v, got: %v", tmDelete, gotNeg.TMDelete)
	}

	// miss.
	idMiss := uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333")
	_, found, err = h.ActiveflowWebhookGet(ctx, idMiss)
	if err != nil {
		t.Fatalf("Could not get the miss entry. err: %v", err)
	}
	if found {
		t.Errorf("Wrong match. expect: miss, got: found")
	}
}

// Test_ActiveflowWebhook_MonotonicGuard verifies the atomic compare-and-set:
// an older-Tm write does NOT overwrite a newer-Tm stored entry, guarding
// against the created-after-deleted resurrection race (design 5.6).
func Test_ActiveflowWebhook_MonotonicGuard(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	id := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")

	tmOld := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	tmNew := time.Date(2026, 6, 10, 2, 0, 0, 0, time.UTC)

	// 1. store a NEW deleted tombstone first (simulating a delete event that
	// arrived before a late created event).
	if err := h.ActiveflowWebhookSetNegative(ctx, id, tmNew, &tmNew, time.Hour); err != nil {
		t.Fatalf("Could not set the newer negative entry. err: %v", err)
	}

	// 2. a late, OLDER positive write must be skipped (resurrection guard).
	wOld := &mwactiveflow.Webhook{
		URI:    "af.test.com",
		Method: webhook.MethodTypePOST,
		Tm:     tmOld,
	}
	if err := h.ActiveflowWebhookSet(ctx, id, wOld, time.Hour); err != nil {
		t.Fatalf("Could not run the older positive write. err: %v", err)
	}

	got, found, err := h.ActiveflowWebhookGet(ctx, id)
	if err != nil {
		t.Fatalf("Could not get the entry. err: %v", err)
	}
	if !found {
		t.Fatalf("Wrong match. expect: found, got: miss")
	}
	if got.IsPositive() {
		t.Errorf("Wrong match. older write must NOT overwrite newer tombstone, got positive: %v", got)
	}
	if !got.Tm.Equal(tmNew) {
		t.Errorf("Wrong match. expect tm still: %v, got: %v", tmNew, got.Tm)
	}

	// 3. a NEWER positive write (>= stored Tm) must overwrite.
	tmNewer := time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC)
	wNewer := &mwactiveflow.Webhook{
		URI:    "af.newer.com",
		Method: webhook.MethodTypePOST,
		Tm:     tmNewer,
	}
	if err := h.ActiveflowWebhookSet(ctx, id, wNewer, time.Hour); err != nil {
		t.Fatalf("Could not run the newer positive write. err: %v", err)
	}

	got, found, err = h.ActiveflowWebhookGet(ctx, id)
	if err != nil {
		t.Fatalf("Could not get the entry. err: %v", err)
	}
	if !found {
		t.Fatalf("Wrong match. expect: found, got: miss")
	}
	if !got.IsPositive() || got.URI != "af.newer.com" {
		t.Errorf("Wrong match. newer write must overwrite, got: %v", got)
	}
}

// Test_ActiveflowWebhook_TTLApplied verifies the TTL is honored by the Lua set.
func Test_ActiveflowWebhook_TTLApplied(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	id := uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")
	w := &mwactiveflow.Webhook{
		URI:    "af.test.com",
		Method: webhook.MethodTypePOST,
		Tm:     time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
	}
	if err := h.ActiveflowWebhookSet(ctx, id, w, 2*time.Minute); err != nil {
		t.Fatalf("Could not set the entry. err: %v", err)
	}

	ttl := mr.TTL(activeflowWebhookKey(id))
	if ttl <= 0 || ttl > 2*time.Minute {
		t.Errorf("Wrong match. expect a positive ttl <= 2m, got: %v", ttl)
	}

	// fast-forward past the TTL and confirm the key is gone.
	mr.FastForward(3 * time.Minute)
	_, found, err := h.ActiveflowWebhookGet(ctx, id)
	if err != nil {
		t.Fatalf("Could not get the entry. err: %v", err)
	}
	if found {
		t.Errorf("Wrong match. expect: expired/miss, got: found")
	}
}
