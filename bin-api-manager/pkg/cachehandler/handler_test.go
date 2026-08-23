package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
)

// newTestHandler starts an in-memory Redis server and returns a handler
// connected to it, plus the miniredis instance for TTL manipulation.
func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	return &handler{Cache: client}, mr
}

func Test_ProvisioningTokenSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	token := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	extensionID := uuid.FromStringOrNil("6a1cca9e-80fb-11f0-92cd-2f8ae91ffc6e")
	ttl := 10 * time.Minute

	if errSet := h.ProvisioningTokenSet(ctx, token, extensionID, ttl); errSet != nil {
		t.Fatalf("ProvisioningTokenSet: unexpected error: %v", errSet)
	}

	// The value must be stored under the documented key with the TTL applied.
	key := provisioningTokenKeyPrefix + token
	if !mr.Exists(key) {
		t.Fatalf("expected key %q to exist in redis", key)
	}
	if got := mr.TTL(key); got != ttl {
		t.Errorf("wrong TTL. expect: %v, got: %v", ttl, got)
	}

	res, err := h.ProvisioningTokenGet(ctx, token)
	if err != nil {
		t.Fatalf("ProvisioningTokenGet: unexpected error: %v", err)
	}
	if res != extensionID {
		t.Errorf("wrong extension id. expect: %v, got: %v", extensionID, res)
	}
}

func Test_ProvisioningTokenGet_ExpiredToken(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	token := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	extensionID := uuid.FromStringOrNil("a41cd53c-80fb-11f0-b5c5-53ef4c0f36b8")

	if errSet := h.ProvisioningTokenSet(ctx, token, extensionID, 10*time.Minute); errSet != nil {
		t.Fatalf("ProvisioningTokenSet: unexpected error: %v", errSet)
	}

	// Still resolvable right before expiry.
	mr.FastForward(10*time.Minute - time.Second)
	if _, err := h.ProvisioningTokenGet(ctx, token); err != nil {
		t.Fatalf("ProvisioningTokenGet before expiry: unexpected error: %v", err)
	}

	// Expired after the TTL elapses.
	mr.FastForward(2 * time.Second)
	res, err := h.ProvisioningTokenGet(ctx, token)
	if err == nil {
		t.Errorf("expected error for expired token, got id: %v", res)
	}
	if res != uuid.Nil {
		t.Errorf("expected uuid.Nil for expired token, got: %v", res)
	}
}

func Test_ProvisioningTokenGet_MissingToken(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	res, err := h.ProvisioningTokenGet(ctx, "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Errorf("expected error for missing token, got id: %v", res)
	}
	if res != uuid.Nil {
		t.Errorf("expected uuid.Nil for missing token, got: %v", res)
	}
}

func Test_ProvisioningTokenGet_CorruptValue(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	token := "1111111111111111111111111111111111111111111111111111111111111111"
	if errSet := mr.Set(provisioningTokenKeyPrefix+token, "not-a-uuid"); errSet != nil {
		t.Fatalf("could not seed corrupt value: %v", errSet)
	}

	res, err := h.ProvisioningTokenGet(ctx, token)
	if err == nil {
		t.Errorf("expected error for corrupt value, got id: %v", res)
	}
	if res != uuid.Nil {
		t.Errorf("expected uuid.Nil for corrupt value, got: %v", res)
	}
}
