package cachehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
)

const (
	testID         = "5d2c9e10-0001-11ee-8a11-000000000001"
	testCustomerID = "5d2c9e10-0002-11ee-8a11-000000000002"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h := &handler{
		Addr: mr.Addr(),
		Cache: redis.NewClient(&redis.Options{
			Addr:        mr.Addr(),
			MaxRetries:  -1,
			DialTimeout: 200 * time.Millisecond,
		}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func testIdentity() commonidentity.Identity {
	return commonidentity.Identity{
		ID:         uuid.FromStringOrNil(testID),
		CustomerID: uuid.FromStringOrNil(testCustomerID),
	}
}

func Test_OutplanSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	o := &outplan.Outplan{
		Identity:     testIdentity(),
		Name:         "test outplan",
		DialTimeout:  30000,
		TryInterval:  60000,
		MaxTryCount0: 3,
	}

	if err := h.OutplanSet(ctx, o); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the entry is stored under the type-prefixed key with a 24 hour expiry.
	key := "outplan:" + testID
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect key %s to exist", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.OutplanGet(ctx, o.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(o, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", o, res)
	}
}

func Test_CampaignSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	c := &campaign.Campaign{
		Identity:     testIdentity(),
		Type:         campaign.TypeCall,
		Name:         "test campaign",
		Status:       campaign.StatusRun,
		ServiceLevel: 50,
	}

	if err := h.CampaignSet(ctx, c); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "campaign:" + testID
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect key %s to exist", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.CampaignGet(ctx, c.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(c, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", c, res)
	}
}

func Test_CampaigncallSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	c := &campaigncall.Campaigncall{
		Identity:         testIdentity(),
		CampaignID:       uuid.FromStringOrNil("5d2c9e10-0003-11ee-8a11-000000000003"),
		Status:           campaigncall.StatusDialing,
		DestinationIndex: 1,
		TryCount:         2,
	}

	if err := h.CampaigncallSet(ctx, c); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "campaigncall:" + testID
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect key %s to exist", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.CampaigncallGet(ctx, c.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(c, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", c, res)
	}
}

func Test_Get_miss(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(testID)

	tests := []struct {
		name string
		get  func() (any, error)
	}{
		{"outplan", func() (any, error) { return h.OutplanGet(ctx, id) }},
		{"campaign", func() (any, error) { return h.CampaignGet(ctx, id) }},
		{"campaigncall", func() (any, error) { return h.CampaigncallGet(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.get()
			if !errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
			}
			if !isNilResult(res) {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func isNilResult(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

func Test_Get_corruptedValue(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil(testID)

	tests := []struct {
		name string
		key  string
		get  func() (any, error)
	}{
		{"outplan", "outplan:" + testID, func() (any, error) { return h.OutplanGet(ctx, id) }},
		{"campaign", "campaign:" + testID, func() (any, error) { return h.CampaignGet(ctx, id) }},
		{"campaigncall", "campaigncall:" + testID, func() (any, error) { return h.CampaigncallGet(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := mr.Set(tt.key, "{not a json"); err != nil {
				t.Fatalf("could not prepare the cache. err: %v", err)
			}

			res, err := tt.get()
			if err == nil {
				t.Errorf("Wrong match. expect: unmarshal error, got: nil")
			}
			if errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: not a cache-miss error, got: %v", err)
			}
			if !isNilResult(res) {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

func Test_Set_serverDown(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	mr.Close()

	if err := h.OutplanSet(ctx, &outplan.Outplan{Identity: testIdentity()}); err == nil {
		t.Errorf("Wrong match. outplan expect: error, got: nil")
	}
	if err := h.CampaignSet(ctx, &campaign.Campaign{Identity: testIdentity()}); err == nil {
		t.Errorf("Wrong match. campaign expect: error, got: nil")
	}
	if err := h.CampaigncallSet(ctx, &campaigncall.Campaigncall{Identity: testIdentity()}); err == nil {
		t.Errorf("Wrong match. campaigncall expect: error, got: nil")
	}
}

func Test_Connect(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server is closed, got: nil")
	}
}

func Test_NewHandler(t *testing.T) {
	mr := miniredis.RunT(t)
	password := "test-secret"

	res := NewHandler(mr.Addr(), password, 3)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	if h.Addr != mr.Addr() || h.Password != password || h.DB != 3 || h.Cache == nil {
		t.Errorf("Wrong match. got: addr=%s db=%d cache=%v", h.Addr, h.DB, h.Cache)
	}

	opt := h.Cache.Options()
	if opt.Addr != mr.Addr() || opt.Password != password || opt.DB != 3 {
		t.Errorf("Wrong match. client options: addr=%s db=%d", opt.Addr, opt.DB)
	}
}
