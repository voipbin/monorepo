package ratelimithandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func i1NewHandler(t *testing.T) (RateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return NewHandler(client), mr
}

func Test_Allow_RedisError(t *testing.T) {
	h, mr := i1NewHandler(t)
	mr.Close()

	res, err := h.Allow(context.Background(), "i1-key", Limit{Rate: 1, Burst: 1, Period: time.Second})
	if err == nil {
		t.Errorf("expected redis error, got nil")
	}
	if res != nil {
		t.Errorf("expected nil result on error, got: %+v", res)
	}
}

func Test_Allow_RemainingAndDenied(t *testing.T) {
	h, _ := i1NewHandler(t)
	ctx := context.Background()
	limit := Limit{Rate: 1, Burst: 3, Period: time.Hour}

	// burst of 3: remaining goes 2, 1, 0 and RetryAfter is -1 while allowed.
	for i, expectRemaining := range []int{2, 1, 0} {
		res, err := h.Allow(ctx, "i1-key", limit)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
		if res.Allowed != 1 {
			t.Errorf("call %d: expected Allowed=1, got %d", i, res.Allowed)
		}
		if res.Remaining != expectRemaining {
			t.Errorf("call %d: wrong Remaining. expect: %d, got: %d", i, expectRemaining, res.Remaining)
		}
		if res.RetryAfter != -1 {
			t.Errorf("call %d: expected RetryAfter=-1, got %v", i, res.RetryAfter)
		}
	}

	res, err := h.Allow(ctx, "i1-key", limit)
	if err != nil {
		t.Fatalf("denied call: unexpected error: %v", err)
	}
	if res.Allowed != 0 || res.Remaining != 0 {
		t.Errorf("denied call: expected Allowed=0 Remaining=0, got %+v", res)
	}
	if res.RetryAfter <= 0 || res.RetryAfter > time.Hour {
		t.Errorf("denied call: RetryAfter out of range (0, 1h]: %v", res.RetryAfter)
	}
}
