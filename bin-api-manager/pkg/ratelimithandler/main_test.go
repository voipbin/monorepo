package ratelimithandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

// TestHandler_Allow_MiniredisIntegration verifies the GCRA behavior end to
// end against a real (in-memory) Redis server, confirming that miniredis is
// compatible with redis_rate's Lua script (including its TIME command
// usage) per design doc §4-15.
func TestHandler_Allow_MiniredisIntegration(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() {
		_ = client.Close()
	}()

	h := NewHandler(client)
	ctx := context.Background()
	limit := Limit{Rate: 1, Burst: 1, Period: time.Second}

	res1, err := h.Allow(ctx, "test-key", limit)
	if err != nil {
		t.Fatalf("first Allow: unexpected error: %v", err)
	}
	if res1.Allowed != 1 {
		t.Errorf("first Allow: expected Allowed=1, got %d", res1.Allowed)
	}

	res2, err := h.Allow(ctx, "test-key", limit)
	if err != nil {
		t.Fatalf("second Allow: unexpected error: %v", err)
	}
	if res2.Allowed != 0 {
		t.Errorf("second Allow: expected Allowed=0 (burst exhausted), got %d", res2.Allowed)
	}
	if res2.RetryAfter <= 0 {
		t.Errorf("second Allow: expected positive RetryAfter, got %v", res2.RetryAfter)
	}

	// Independent key is unaffected.
	res3, err := h.Allow(ctx, "other-key", limit)
	if err != nil {
		t.Fatalf("independent key Allow: unexpected error: %v", err)
	}
	if res3.Allowed != 1 {
		t.Errorf("independent key: expected Allowed=1, got %d", res3.Allowed)
	}
}
