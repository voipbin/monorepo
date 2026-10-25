package cachehandler

import (
	"context"
	"testing"
	"time"

	"monorepo/bin-tts-manager/models/streaming"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

func newTestCache(t *testing.T) (*miniredis.Miniredis, *handler) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatal("NewHandler did not return the concrete handler")
	}
	t.Cleanup(func() {
		_ = h.Cache.Close()
	})

	return mr, h
}

func Test_StreamingSetGet(t *testing.T) {
	mr, h := newTestCache(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("b6dcbc72-5af5-11f0-b0a7-fbe8aae41529")
	src := &streaming.Streaming{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("b685d36c-5af5-11f0-b36b-2be7103890ae"),
		},
		ReferenceType: streaming.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("b6b3ba5c-5af5-11f0-bf7e-438fa6d3dba0"),
		Language:      "en-US",
		Gender:        streaming.GenderFemale,
		Direction:     streaming.DirectionOutgoing,
	}

	if err := h.StreamingSet(ctx, src); err != nil {
		t.Fatalf("StreamingSet failed: %v", err)
	}

	key := "tts:streaming:" + id.String()
	if !mr.Exists(key) {
		t.Fatalf("expected the key %s to exist", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong ttl. expect: %v, got: %v", 24*time.Hour, ttl)
	}

	got, err := h.StreamingGet(ctx, id)
	if err != nil {
		t.Fatalf("StreamingGet failed: %v", err)
	}

	if got.ID != src.ID || got.CustomerID != src.CustomerID ||
		got.ReferenceType != src.ReferenceType || got.ReferenceID != src.ReferenceID ||
		got.Language != src.Language || got.Gender != src.Gender || got.Direction != src.Direction {
		t.Errorf("Wrong streaming.\nexpect: %+v\ngot: %+v", src, got)
	}
}

func Test_StreamingGet_missingKey(t *testing.T) {
	_, h := newTestCache(t)

	got, err := h.StreamingGet(context.Background(), uuid.FromStringOrNil("b6dcbc72-5af5-11f0-b0a7-fbe8aae41529"))
	if got != nil || err != redis.Nil {
		t.Errorf("Wrong result. expect redis.Nil, got: %v, %v", got, err)
	}
}

func Test_StreamingGet_corruptedValue(t *testing.T) {
	mr, h := newTestCache(t)

	id := uuid.FromStringOrNil("b6dcbc72-5af5-11f0-b0a7-fbe8aae41529")
	if err := mr.Set("tts:streaming:"+id.String(), "{not json"); err != nil {
		t.Fatalf("could not seed the value: %v", err)
	}

	got, err := h.StreamingGet(context.Background(), id)
	if got != nil || err == nil {
		t.Errorf("expected a parse error, got: %v, %v", got, err)
	}
	if err == redis.Nil {
		t.Errorf("a corrupted value must not look like a missing key")
	}
}

func Test_StreamingSet_serverUnavailable(t *testing.T) {
	mr, h := newTestCache(t)

	mr.Close()

	err := h.StreamingSet(context.Background(), &streaming.Streaming{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}})
	if err == nil {
		t.Errorf("expected an error when the server is down")
	}
}

func Test_Connect(t *testing.T) {
	mr, h := newTestCache(t)

	if err := h.Connect(); err != nil {
		t.Errorf("expected a successful connection, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("expected an error after the server was stopped")
	}
}

func Test_NewHandler_fields(t *testing.T) {
	mr := miniredis.RunT(t)

	password := "test-secret"
	h, ok := NewHandler(mr.Addr(), password, 3).(*handler)
	if !ok {
		t.Fatal("NewHandler did not return the concrete handler")
	}
	defer func() {
		_ = h.Cache.Close()
	}()

	if h.Addr != mr.Addr() || h.Password != password || h.DB != 3 {
		t.Errorf("Wrong connection settings. addr: %s, db: %d", h.Addr, h.DB)
	}
	if h.Cache == nil || h.Locker == nil || h.mapMutex == nil {
		t.Errorf("Expected the clients to be initialized. cache: %v, locker: %v", h.Cache, h.Locker)
	}
}
