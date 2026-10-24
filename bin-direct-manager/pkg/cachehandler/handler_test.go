package cachehandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-direct-manager/models/direct"
)

func newTestCache(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatalf("NewHandler did not return *handler")
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func newTestDirect() *direct.Direct {
	return &direct.Direct{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("27d26bf2-2a01-11ee-82a4-63ea4f4f7211"),
			CustomerID: uuid.FromStringOrNil("a082d59c-2a00-11ee-8fb1-8bbf141432f6"),
		},
		ResourceType: "extension",
		ResourceID:   uuid.FromStringOrNil("c31676f0-4e69-11ec-afe3-77ba49fae527"),
		Hash:         "direct.abcdef123456",
	}
}

func Test_DirectSetByHash_DirectGetByHash(t *testing.T) {
	h, mr := newTestCache(t)
	ctx := context.Background()
	d := newTestDirect()

	if err := h.DirectSetByHash(ctx, d.Hash, d); err != nil {
		t.Fatalf("DirectSetByHash() error = %v", err)
	}

	// the entry lives under the hash key for one day
	key := "direct:hash:" + d.Hash
	if !mr.Exists(key) {
		t.Fatalf("key %q was not stored", key)
	}
	if got := mr.TTL(key); got != 24*time.Hour {
		t.Errorf("TTL of %q = %v, want %v", key, got, 24*time.Hour)
	}

	res, err := h.DirectGetByHash(ctx, d.Hash)
	if err != nil {
		t.Fatalf("DirectGetByHash() error = %v", err)
	}
	if !reflect.DeepEqual(res, d) {
		t.Errorf("DirectGetByHash() = %v, want %v", res, d)
	}
}

func Test_DirectGetByHash_miss(t *testing.T) {
	h, _ := newTestCache(t)

	res, err := h.DirectGetByHash(context.Background(), "direct.000000000000")
	if !errors.Is(err, redis.Nil) {
		t.Errorf("DirectGetByHash() error = %v, want redis.Nil", err)
	}
	if res != nil {
		t.Errorf("DirectGetByHash() = %v, want nil", res)
	}
}

func Test_DirectGetByHash_corruptedValue(t *testing.T) {
	h, mr := newTestCache(t)

	if err := mr.Set("direct:hash:direct.abcdef123456", "{not json"); err != nil {
		t.Fatalf("could not seed the value: %v", err)
	}

	res, err := h.DirectGetByHash(context.Background(), "direct.abcdef123456")
	if err == nil {
		t.Fatalf("DirectGetByHash() error = nil, want an unmarshal error")
	}
	if errors.Is(err, redis.Nil) {
		t.Errorf("DirectGetByHash() error = %v, a corrupted value must not look like a miss", err)
	}
	if res != nil {
		t.Errorf("DirectGetByHash() = %v, want nil", res)
	}
}

func Test_DirectDeleteByHash(t *testing.T) {
	h, mr := newTestCache(t)
	ctx := context.Background()
	d := newTestDirect()
	other := newTestDirect()
	other.Hash = "direct.fedcba654321"

	if err := h.DirectSetByHash(ctx, d.Hash, d); err != nil {
		t.Fatalf("could not seed: %v", err)
	}
	if err := h.DirectSetByHash(ctx, other.Hash, other); err != nil {
		t.Fatalf("could not seed: %v", err)
	}

	if err := h.DirectDeleteByHash(ctx, d.Hash); err != nil {
		t.Fatalf("DirectDeleteByHash() error = %v", err)
	}
	if mr.Exists("direct:hash:" + d.Hash) {
		t.Errorf("deleted key still exists")
	}
	if !mr.Exists("direct:hash:" + other.Hash) {
		t.Errorf("deleting one hash removed another hash")
	}

	// deleting a key that is not there is not an error
	if err := h.DirectDeleteByHash(ctx, d.Hash); err != nil {
		t.Errorf("DirectDeleteByHash() on a missing key error = %v", err)
	}
}

func Test_cache_serverDown(t *testing.T) {
	h, mr := newTestCache(t)
	ctx := context.Background()
	d := newTestDirect()
	mr.Close()

	if err := h.DirectSetByHash(ctx, d.Hash, d); err == nil {
		t.Errorf("DirectSetByHash() error = nil, want a connection error")
	}
	if _, err := h.DirectGetByHash(ctx, d.Hash); err == nil || errors.Is(err, redis.Nil) {
		t.Errorf("DirectGetByHash() error = %v, want a connection error", err)
	}
	if err := h.DirectDeleteByHash(ctx, d.Hash); err == nil {
		t.Errorf("DirectDeleteByHash() error = nil, want a connection error")
	}
}

func Test_setSerialize_unmarshalable(t *testing.T) {
	h, mr := newTestCache(t)

	if err := h.setSerialize(context.Background(), "k", make(chan int)); err == nil {
		t.Errorf("setSerialize() error = nil, want a marshal error")
	}
	if mr.Exists("k") {
		t.Errorf("a value that failed to marshal was stored")
	}
}

func Test_Connect(t *testing.T) {
	mr := miniredis.RunT(t)

	h := NewHandler(mr.Addr(), "", 0)
	if err := h.Connect(); err != nil {
		t.Errorf("Connect() error = %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Connect() after the server stopped error = nil, want an error")
	}
}

func Test_NewHandler_password(t *testing.T) {
	mr := miniredis.RunT(t)
	right := strings.Repeat("r", 12)
	wrong := strings.Repeat("w", 12)
	mr.RequireAuth(right)

	if err := NewHandler(mr.Addr(), wrong, 0).Connect(); err == nil {
		t.Errorf("Connect() with a wrong password error = nil, want an error")
	}
	if err := NewHandler(mr.Addr(), right, 0).Connect(); err != nil {
		t.Errorf("Connect() with the right password error = %v", err)
	}
}

func Test_NewHandler_database(t *testing.T) {
	mr := miniredis.RunT(t)
	h := NewHandler(mr.Addr(), "", 3)
	d := newTestDirect()

	if err := h.DirectSetByHash(context.Background(), d.Hash, d); err != nil {
		t.Fatalf("DirectSetByHash() error = %v", err)
	}

	if !mr.DB(3).Exists("direct:hash:" + d.Hash) {
		t.Errorf("the entry was not written to database 3")
	}
	if mr.DB(0).Exists("direct:hash:" + d.Hash) {
		t.Errorf("the entry was written to database 0 instead of database 3")
	}
}
