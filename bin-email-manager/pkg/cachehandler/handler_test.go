package cachehandler

import (
	"context"
	"net"
	"testing"
	"time"

	"monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

// newUnreachableHandler returns a handler whose cache client points at a loopback port
// that nothing listens on, with retries disabled so failures are immediate.
func newUnreachableHandler(t *testing.T) *handler {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return &handler{
		Addr: addr,
		Cache: redis.NewClient(&redis.Options{
			Addr:        addr,
			MaxRetries:  -1,
			DialTimeout: 200 * time.Millisecond,
		}),
	}
}

func Test_NewHandler(t *testing.T) {
	res := NewHandler("127.0.0.1:6379", "synthetic-password", 3)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.Addr != "127.0.0.1:6379" || h.Password != "synthetic-password" || h.DB != 3 {
		t.Errorf("Wrong settings. addr: %s, db: %d", h.Addr, h.DB)
	}
	if h.Cache == nil || h.Locker == nil || h.mapMutex == nil {
		t.Errorf("Expected the cache, the locker and the mutex map to be initialized")
	}
}

func Test_unreachableCache(t *testing.T) {
	h := newUnreachableHandler(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	id := uuid.FromStringOrNil("5e8f0f0a-8888-4888-8888-000000000001")

	if err := h.Connect(); err == nil {
		t.Errorf("Connect: expected an error")
	}

	if err := h.EmailSet(ctx, &email.Email{}); err == nil {
		t.Errorf("EmailSet: expected an error")
	}

	res, err := h.EmailGet(ctx, id)
	if err == nil {
		t.Errorf("EmailGet: expected an error")
	}
	if res != nil {
		t.Errorf("EmailGet: expected a nil result. got: %v", res)
	}
}
