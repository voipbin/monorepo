package cachehandler

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-number-manager/models/number"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatalf("NewHandler() did not return *handler")
	}
	// no client retries so that a stopped server fails fast
	_ = h.Cache.Close()
	h.Cache = redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() {
		_ = h.Cache.Close()
	})

	return h, mr
}

func newTestNumber() *number.Number {
	return &number.Number{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50"),
			CustomerID: uuid.FromStringOrNil("6a7b8c9d-2222-4b5c-8d9e-1a2b3c4d5e6f"),
		},
		Number:       "+15550100001",
		Name:         "test name",
		Detail:       "test detail",
		ProviderName: number.ProviderNameTelnyx,
		Status:       number.StatusActive,
		T38Enabled:   true,
	}
}

func Test_NewHandler(t *testing.T) {
	mr := miniredis.RunT(t)
	addr := mr.Addr()

	h, ok := NewHandler(addr, "", 3).(*handler)
	if !ok {
		t.Fatalf("NewHandler() did not return *handler")
	}
	defer func() {
		_ = h.Cache.Close()
	}()

	if h.Addr != addr {
		t.Errorf("Wrong Addr. expect: %s, got: %s", addr, h.Addr)
	}
	if h.DB != 3 {
		t.Errorf("Wrong DB. expect: 3, got: %d", h.DB)
	}
	if h.Cache == nil {
		t.Errorf("Wrong Cache. expect: not nil, got: nil")
	}
}

func Test_Connect(t *testing.T) {
	t.Run("reachable server", func(t *testing.T) {
		h, _ := newTestHandler(t)

		if err := h.Connect(); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("closed server", func(t *testing.T) {
		h, mr := newTestHandler(t)
		mr.Close()

		if err := h.Connect(); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_NumberSet_NumberGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	n := newTestNumber()
	if err := h.NumberSet(ctx, n); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// both keys are written
	keyID := "number:5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50"
	keyNumber := "number-number:+15550100001"
	for _, key := range []string{keyID, keyNumber} {
		if !mr.Exists(key) {
			t.Errorf("Wrong match. expect: key %s exists, got: missing", key)
		}
		if ttl := mr.TTL(key); ttl != 24*time.Hour {
			t.Errorf("Wrong ttl of %s. expect: 24h, got: %v", key, ttl)
		}
	}

	res, err := h.NumberGet(ctx, n.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(n, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", n, res)
	}

	res, err = h.NumberGetByNumber(ctx, "+15550100001")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(n, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", n, res)
	}
}

func Test_NumberSetByNumber_NumberDelByNumber(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	n := newTestNumber()
	if err := h.NumberSetByNumber(ctx, n); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// only the number key is written
	if !mr.Exists("number-number:+15550100001") {
		t.Errorf("Wrong match. expect: number key exists, got: missing")
	}
	if mr.Exists("number:5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50") {
		t.Errorf("Wrong match. expect: id key missing, got: exists")
	}

	res, err := h.NumberGetByNumber(ctx, "+15550100001")
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(n, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", n, res)
	}

	if err := h.NumberDelByNumber(ctx, "+15550100001"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists("number-number:+15550100001") {
		t.Errorf("Wrong match. expect: number key deleted, got: exists")
	}

	if _, err := h.NumberGetByNumber(ctx, "+15550100001"); err == nil {
		t.Errorf("Wrong match. expect: error after delete, got: nil")
	}
}

func Test_NumberDel(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	n := newTestNumber()
	if err := h.NumberSet(ctx, n); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if err := h.NumberDel(ctx, n.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if mr.Exists("number:5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50") {
		t.Errorf("Wrong match. expect: id key deleted, got: exists")
	}
	// the number key is untouched by NumberDel
	if !mr.Exists("number-number:+15550100001") {
		t.Errorf("Wrong match. expect: number key kept, got: missing")
	}

	if _, err := h.NumberGet(ctx, n.ID); err == nil {
		t.Errorf("Wrong match. expect: error after delete, got: nil")
	}
}

func Test_Get_missing_key(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	if _, err := h.NumberGet(ctx, uuid.FromStringOrNil("5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50")); err != redis.Nil {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}
	if _, err := h.NumberGetByNumber(ctx, "+15550100002"); err != redis.Nil {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}
}

func Test_Get_corrupted_json(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	if err := mr.Set("number:5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50", "{not-json"); err != nil {
		t.Fatalf("could not seed the corrupted value: %v", err)
	}
	if err := mr.Set("number-number:+15550100003", "{not-json"); err != nil {
		t.Fatalf("could not seed the corrupted value: %v", err)
	}

	if _, err := h.NumberGet(ctx, uuid.FromStringOrNil("5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50")); err == nil {
		t.Errorf("Wrong match. expect: unmarshal error, got: nil")
	}
	if _, err := h.NumberGetByNumber(ctx, "+15550100003"); err == nil {
		t.Errorf("Wrong match. expect: unmarshal error, got: nil")
	}
}

func Test_server_down(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	n := newTestNumber()
	mr.Close()

	if err := h.NumberSet(ctx, n); err == nil {
		t.Errorf("NumberSet: expect: error, got: nil")
	}
	if err := h.NumberSetByNumber(ctx, n); err == nil {
		t.Errorf("NumberSetByNumber: expect: error, got: nil")
	}
	if _, err := h.NumberGet(ctx, n.ID); err == nil {
		t.Errorf("NumberGet: expect: error, got: nil")
	}
	if _, err := h.NumberGetByNumber(ctx, n.Number); err == nil {
		t.Errorf("NumberGetByNumber: expect: error, got: nil")
	}
	if err := h.NumberDel(ctx, n.ID); err == nil {
		t.Errorf("NumberDel: expect: error, got: nil")
	}
	if err := h.NumberDelByNumber(ctx, n.Number); err == nil {
		t.Errorf("NumberDelByNumber: expect: error, got: nil")
	}
}

// failKeyHook fails the commands whose first argument contains the given substring.
type failKeyHook struct {
	substr string
}

func (f failKeyHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	for _, arg := range cmd.Args() {
		if s, ok := arg.(string); ok && strings.Contains(s, f.substr) {
			return ctx, redis.ErrClosed
		}
	}
	return ctx, nil
}

func (f failKeyHook) AfterProcess(ctx context.Context, cmd redis.Cmder) error {
	return nil
}

func (f failKeyHook) BeforeProcessPipeline(ctx context.Context, cmds []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (f failKeyHook) AfterProcessPipeline(ctx context.Context, cmds []redis.Cmder) error {
	return nil
}

func Test_NumberSet_second_key_failure(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	h.Cache.AddHook(failKeyHook{substr: "number-number:"})

	n := newTestNumber()
	if err := h.NumberSet(ctx, n); err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}

	// the id key was written before the number key failed
	if !mr.Exists("number:5f2c1a64-1111-4a6e-9a31-0b1c2d3e4f50") {
		t.Errorf("Wrong match. expect: id key exists, got: missing")
	}
	if mr.Exists("number-number:+15550100001") {
		t.Errorf("Wrong match. expect: number key missing, got: exists")
	}
}
