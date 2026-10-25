package cachehandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	s := miniredis.RunT(t)
	return &handler{
		Cache: redis.NewClient(&redis.Options{Addr: s.Addr()}),
	}, s
}

func seedJSON(t *testing.T, s *miniredis.Miniredis, key string, v any) {
	t.Helper()

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal the seed: %v", err)
	}
	if err := s.Set(key, string(b)); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
}

func Test_AccountGet(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	h, s := newTestHandler(t)
	seeded := &account.Account{
		Identity:      commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:          "test account",
		BalanceCredit: 1234567,
		BalanceToken:  89,
	}
	seedJSON(t, s, "billing:account:"+id.String(), seeded)

	res, err := h.AccountGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id || res.CustomerID != customerID || res.Name != "test account" || res.BalanceCredit != 1234567 || res.BalanceToken != 89 {
		t.Errorf("Wrong match. got: %+v", res)
	}

	// a different id is a miss
	if _, err := h.AccountGet(ctx, uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")); err != redis.Nil {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}

	// a corrupt value is an error that is not a miss
	if err := s.Set("billing:account:"+id.String(), "{not json"); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
	res, err = h.AccountGet(ctx, id)
	if err == nil || err == redis.Nil || res != nil {
		t.Errorf("Wrong match. expect: unmarshal error and nil result, got: %v, %v", err, res)
	}
}

func Test_AccountGetByCustomerID(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	h, s := newTestHandler(t)
	seedJSON(t, s, "billing:account-customer_id:"+customerID.String(), &account.Account{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:     "by customer",
	})

	res, err := h.AccountGetByCustomerID(ctx, customerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id || res.Name != "by customer" {
		t.Errorf("Wrong match. got: %+v", res)
	}

	if _, err := h.AccountGetByCustomerID(ctx, id); err != redis.Nil {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}

	if err := s.Set("billing:account-customer_id:"+customerID.String(), "[1,2"); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
	if res, err := h.AccountGetByCustomerID(ctx, customerID); err == nil || err == redis.Nil || res != nil {
		t.Errorf("Wrong match. expect: unmarshal error and nil result, got: %v, %v", err, res)
	}
}

func Test_AccountSet(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")

	h, s := newTestHandler(t)
	data := &account.Account{
		Identity:      commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:          "stored account",
		BalanceCredit: 42,
	}

	if err := h.AccountSet(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "billing:account:" + id.String()
	raw, err := s.Get(key)
	if err != nil {
		t.Fatalf("Wrong match. expect the primary key to exist, got: %v", err)
	}
	var stored account.Account
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("Wrong match. stored value is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(&stored, data) {
		t.Errorf("Wrong match.\nexpect: %+v\ngot:    %+v", data, &stored)
	}
	if ttl := s.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: %v, got: %v", 24*time.Hour, ttl)
	}

	// the stored value is readable through the getter
	res, err := h.AccountGet(ctx, id)
	if err != nil || res.Name != "stored account" {
		t.Errorf("Wrong match. got: %+v, %v", res, err)
	}
}

func Test_AccountSet_serverDown(t *testing.T) {
	h, s := newTestHandler(t)
	s.Close()

	err := h.AccountSet(context.Background(), &account.Account{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")},
	})
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_BillingGet(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	h, s := newTestHandler(t)
	seedJSON(t, s, "billing:billing:"+id.String(), &billing.Billing{
		Identity:     commonidentity.Identity{ID: id},
		AmountCredit: -500,
		Status:       billing.StatusEnd,
	})

	res, err := h.BillingGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id || res.AmountCredit != -500 || res.Status != billing.StatusEnd {
		t.Errorf("Wrong match. got: %+v", res)
	}

	if _, err := h.BillingGet(ctx, uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")); err != redis.Nil {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}

	if err := s.Set("billing:billing:"+id.String(), "oops"); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
	if res, err := h.BillingGet(ctx, id); err == nil || err == redis.Nil || res != nil {
		t.Errorf("Wrong match. expect: unmarshal error and nil result, got: %v, %v", err, res)
	}
}

func Test_BillingGetByReferenceID(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")
	referenceID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006")

	h, s := newTestHandler(t)
	seedJSON(t, s, "billing:billing-reference_id:"+referenceID.String(), &billing.Billing{
		Identity:    commonidentity.Identity{ID: id},
		ReferenceID: referenceID,
	})

	res, err := h.BillingGetByReferenceID(ctx, referenceID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id || res.ReferenceID != referenceID {
		t.Errorf("Wrong match. got: %+v", res)
	}

	if _, err := h.BillingGetByReferenceID(ctx, id); err != redis.Nil {
		t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
	}

	if err := s.Set("billing:billing-reference_id:"+referenceID.String(), "}"); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
	if res, err := h.BillingGetByReferenceID(ctx, referenceID); err == nil || err == redis.Nil || res != nil {
		t.Errorf("Wrong match. expect: unmarshal error and nil result, got: %v, %v", err, res)
	}
}

func Test_BillingSet(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")
	referenceID := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000006")

	h, s := newTestHandler(t)
	data := &billing.Billing{
		Identity:     commonidentity.Identity{ID: id},
		ReferenceID:  referenceID,
		AmountCredit: -700,
	}

	if err := h.BillingSet(ctx, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	for _, key := range []string{
		"billing:billing:" + id.String(),
		"billing:billing-reference_id:" + referenceID.String(),
	} {
		raw, err := s.Get(key)
		if err != nil {
			t.Fatalf("Wrong match. expect key %s to exist, got: %v", key, err)
		}
		var stored billing.Billing
		if err := json.Unmarshal([]byte(raw), &stored); err != nil {
			t.Fatalf("Wrong match. stored value is not valid JSON: %v", err)
		}
		if !reflect.DeepEqual(&stored, data) {
			t.Errorf("Wrong match for %s.\nexpect: %+v\ngot:    %+v", key, data, &stored)
		}
		if ttl := s.TTL(key); ttl != 24*time.Hour {
			t.Errorf("Wrong match for %s. expect ttl: %v, got: %v", key, 24*time.Hour, ttl)
		}
	}

	byID, err := h.BillingGet(ctx, id)
	if err != nil || byID.AmountCredit != -700 {
		t.Errorf("Wrong match. got: %+v, %v", byID, err)
	}
	byRef, err := h.BillingGetByReferenceID(ctx, referenceID)
	if err != nil || byRef.ID != id {
		t.Errorf("Wrong match. got: %+v, %v", byRef, err)
	}
}

func Test_BillingSet_serverDown(t *testing.T) {
	h, s := newTestHandler(t)
	s.Close()

	if err := h.BillingSet(context.Background(), &billing.Billing{}); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_setSerialize_unmarshalable(t *testing.T) {
	h, s := newTestHandler(t)

	// a channel cannot be encoded as JSON, so nothing may be written
	if err := h.setSerialize(context.Background(), "billing:test", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: marshal error, got: nil")
	}
	if s.Exists("billing:test") {
		t.Errorf("Wrong match. the key must not be written when marshaling fails")
	}
}

func Test_Connect(t *testing.T) {
	s := miniredis.RunT(t)
	h := NewHandler(s.Addr(), "", 0)

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	s.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server is closed, got: nil")
	}
}

func Test_NewHandler(t *testing.T) {
	h := NewHandler("cache.invalid:6379", "", 3)

	tmp, ok := h.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", h)
	}
	if tmp.Addr != "cache.invalid:6379" || tmp.DB != 3 || tmp.Password != "" || tmp.Cache == nil {
		t.Errorf("Wrong match. got: %+v", tmp)
	}
	if got := tmp.Cache.Options().DB; got != 3 {
		t.Errorf("Wrong match. expect client db: 3, got: %d", got)
	}
}
