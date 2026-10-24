package cachehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/route"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Could not start miniredis. err: %v", err)
	}
	t.Cleanup(mr.Close)

	h := &handler{
		Addr:  mr.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func Test_ProviderSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	tmCreate := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	data := &provider.Provider{
		ID:          uuid.FromStringOrNil("b3a0a6c2-0000-4000-8000-000000000001"),
		Type:        provider.TypeSIP,
		Hostname:    "sip.example.invalid",
		TechPrefix:  "pre",
		TechHeaders: map[string]string{"X-Test": "1"},
		Name:        "test provider",
		Detail:      "test detail",
		TMCreate:    &tmCreate,
	}

	if err := h.ProviderSet(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "provider:b3a0a6c2-0000-4000-8000-000000000001"
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect: key %s exists, got: keys %v", key, mr.Keys())
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect: 24h, got: %v", ttl)
	}

	res, err := h.ProviderGet(ctx, data.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, data) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", data, res)
	}
}

func Test_ProviderGet_notFound(t *testing.T) {
	h, _ := newTestHandler(t)

	res, err := h.ProviderGet(context.Background(), uuid.FromStringOrNil("b3a0a6c2-0000-4000-8000-000000000002"))
	if !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

func Test_ProviderGet_corruptedValue(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := mr.Set("provider:b3a0a6c2-0000-4000-8000-000000000003", "{not-json"); err != nil {
		t.Fatalf("Could not seed the value. err: %v", err)
	}

	res, err := h.ProviderGet(context.Background(), uuid.FromStringOrNil("b3a0a6c2-0000-4000-8000-000000000003"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

func Test_RouteSetGetDelete(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	data := &route.Route{
		ID:         uuid.FromStringOrNil("c4b1b7d3-0000-4000-8000-000000000001"),
		CustomerID: uuid.FromStringOrNil("c4b1b7d3-0000-4000-8000-0000000000aa"),
		Name:       "test route",
		ProviderID: uuid.FromStringOrNil("c4b1b7d3-0000-4000-8000-0000000000bb"),
		Priority:   3,
		Target:     "82",
	}

	if err := h.RouteSet(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "route:c4b1b7d3-0000-4000-8000-000000000001"
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect: key %s exists, got: keys %v", key, mr.Keys())
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect: 24h, got: %v", ttl)
	}

	res, err := h.RouteGet(ctx, data.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, data) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", data, res)
	}

	if err := h.RouteDelete(ctx, data.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists(key) {
		t.Errorf("Wrong match. expect: key %s deleted, got: still exists", key)
	}

	if _, err := h.RouteGet(ctx, data.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: redis.Nil after delete, got: %v", err)
	}
}

func Test_RouteGet_corruptedValue(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := mr.Set("route:c4b1b7d3-0000-4000-8000-000000000002", "[1,2"); err != nil {
		t.Fatalf("Could not seed the value. err: %v", err)
	}

	res, err := h.RouteGet(context.Background(), uuid.FromStringOrNil("c4b1b7d3-0000-4000-8000-000000000002"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: ok")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

func Test_serverDown(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mr.Close()

	id := uuid.FromStringOrNil("d5c2c8e4-0000-4000-8000-000000000001")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"provider set", func() error { return h.ProviderSet(ctx, &provider.Provider{ID: id}) }},
		{"provider get", func() error { _, err := h.ProviderGet(ctx, id); return err }},
		{"route set", func() error { return h.RouteSet(ctx, &route.Route{ID: id}) }},
		{"route get", func() error { _, err := h.RouteGet(ctx, id); return err }},
		{"route delete", func() error { return h.RouteDelete(ctx, id) }},
		{"connect", func() error { return h.Connect() }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_NewHandler_connect(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Could not start miniredis. err: %v", err)
	}
	defer mr.Close()

	password := "synthetic-test-secret"
	mr.RequireAuth(password)

	tests := []struct {
		name      string
		password  string
		expectErr bool
	}{
		{"correct password", password, false},
		{"wrong password", "wrong-synthetic-secret", true},
		{"no password", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(mr.Addr(), tt.password, 0)

			err := h.Connect()
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_NewHandler_roundTrip(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Could not start miniredis. err: %v", err)
	}
	defer mr.Close()

	h := NewHandler(mr.Addr(), "", 2)
	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	data := &route.Route{
		ID:     uuid.FromStringOrNil("e6d3d9f5-0000-4000-8000-000000000001"),
		Name:   "round trip",
		Target: "all",
	}
	if err := h.RouteSet(context.Background(), data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the configured database index must be the one that holds the key
	mr.Select(2)
	if !mr.Exists("route:e6d3d9f5-0000-4000-8000-000000000001") {
		t.Errorf("Wrong match. expect: key in database 2, got: not found")
	}
	mr.Select(0)
	if mr.Exists("route:e6d3d9f5-0000-4000-8000-000000000001") {
		t.Errorf("Wrong match. expect: key absent in database 0, got: found")
	}
}
