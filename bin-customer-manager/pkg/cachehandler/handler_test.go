package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	s := miniredis.RunT(t)
	return &handler{
		Cache: redis.NewClient(&redis.Options{Addr: s.Addr()}),
	}, s
}

func Test_ResendCooldownAcquire(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")

	ok, err := h.ResendCooldownAcquire(ctx, customerID, time.Minute)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !ok {
		t.Errorf("Wrong match. expect: true, got: false")
	}

	ok, err = h.ResendCooldownAcquire(ctx, customerID, time.Minute)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if ok {
		t.Errorf("Wrong match. expect: false, got: true")
	}
}

func Test_ResendCountIncr_ArmsTTL(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")

	n, err := h.ResendCountIncr(ctx, customerID, time.Hour)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if n != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", n)
	}

	key := resendCountKeyPrefix + customerID.String()
	if s.TTL(key) <= 0 {
		t.Errorf("Wrong match. expect: positive ttl, got: %v", s.TTL(key))
	}
}

func Test_ResendCountIncr_RepairsMissingTTL(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	key := resendCountKeyPrefix + customerID.String()

	// simulate the crash window: the counter exists with no expiry, which would
	// otherwise bar this customer from resending forever
	if err := s.Set(key, "3"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if s.TTL(key) != 0 {
		t.Fatalf("Wrong match. expect: no ttl, got: %v", s.TTL(key))
	}

	n, err := h.ResendCountIncr(ctx, customerID, time.Hour)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if n != 4 {
		t.Errorf("Wrong match. expect: 4, got: %d", n)
	}
	if s.TTL(key) <= 0 {
		t.Errorf("Wrong match. expect: repaired ttl, got: %v", s.TTL(key))
	}
}
