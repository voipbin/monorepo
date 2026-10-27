package cachehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
)

// failHook makes the given redis commands fail before they reach the server.
type failHook struct {
	fail map[string]error
}

func (h *failHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if err, ok := h.fail[cmd.Name()]; ok {
		return ctx, err
	}
	return ctx, nil
}

func (h *failHook) AfterProcess(_ context.Context, _ redis.Cmder) error {
	return nil
}

func (h *failHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (h *failHook) AfterProcessPipeline(_ context.Context, _ []redis.Cmder) error {
	return nil
}

func Test_ResendCountIncr_commandErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	errInjected := errors.New("injected failure")

	tests := []struct {
		name string

		// number of increments that succeed before the call under test
		preIncr int
		// remove the expiry of the key after the pre-increments
		clearTTL bool
		failCmd  string

		expectN int64
	}{
		{
			name:    "incr fails, no count is returned",
			failCmd: "incr",
			expectN: 0,
		},
		{
			name:    "expire fails on the first increment",
			failCmd: "expire",
			expectN: 1,
		},
		{
			name:    "ttl lookup fails on a later increment",
			preIncr: 1,
			failCmd: "ttl",
			expectN: 2,
		},
		{
			name:     "expire fails while repairing a missing ttl",
			preIncr:  1,
			clearTTL: true,
			failCmd:  "expire",
			expectN:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, s := newTestHandler(t)
			ctx := context.Background()

			key := resendCountKeyPrefix + customerID.String()
			for i := 0; i < tt.preIncr; i++ {
				if _, err := h.ResendCountIncr(ctx, customerID, time.Hour); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
			}
			if tt.clearTTL {
				if err := s.Set(key, "1"); err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if s.TTL(key) != 0 {
					t.Fatalf("Wrong match. expect: no ttl, got: %v", s.TTL(key))
				}
			}

			h.Cache.AddHook(&failHook{fail: map[string]error{tt.failCmd: errInjected}})

			n, err := h.ResendCountIncr(ctx, customerID, time.Hour)
			if !errors.Is(err, errInjected) {
				t.Errorf("Wrong match. expect: %v, got: %v", errInjected, err)
			}
			if n != tt.expectN {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.expectN, n)
			}
		})
	}
}

func Test_setSerialize_unserializableValue(t *testing.T) {
	h, s := newTestHandler(t)

	err := h.setSerialize(context.Background(), "key", make(chan int))
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if s.Exists("key") {
		t.Errorf("Wrong match. expect: nothing is written, got: key exists")
	}
}
