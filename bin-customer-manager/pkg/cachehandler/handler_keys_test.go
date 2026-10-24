package cachehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/models/customer"
)

func Test_CustomerSetGet(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	c := &customer.Customer{
		ID:     id,
		Name:   "cache customer",
		Email:  "cache@example.com",
		Status: customer.StatusActive,
	}

	if _, err := h.CustomerGet(ctx, id); err == nil {
		t.Errorf("Wrong match. expect: miss error, got: nil")
	}

	if err := h.CustomerSet(ctx, c); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "customer:" + id.String()
	if ttl := s.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.CustomerGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, c) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", c, res)
	}

	// a corrupted cache entry must surface as an error, not as a zero customer
	if err := s.Set(key, "{not json"); err != nil {
		t.Fatalf("could not corrupt the key: %v", err)
	}
	if res, err := h.CustomerGet(ctx, id); err == nil {
		t.Errorf("Wrong match. expect: unmarshal error, got result: %v", res)
	}
}

func Test_AccesskeySetGet(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("3ab1c2d3-21b3-4ca7-97ca-069729c87974")
	a := &accesskey.Accesskey{
		ID:         id,
		CustomerID: uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974"),
		Name:       "cache key",
	}

	if _, err := h.AccesskeyGet(ctx, id); err == nil {
		t.Errorf("Wrong match. expect: miss error, got: nil")
	}

	if err := h.AccesskeySet(ctx, a); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "customer_accesskey:" + id.String()
	if ttl := s.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.AccesskeyGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, a) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", a, res)
	}
}

func Test_EmailVerifyToken(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")

	if _, err := h.EmailVerifyTokenGet(ctx, "missing"); err == nil {
		t.Errorf("Wrong match. expect: not found error, got: nil")
	}

	if err := h.EmailVerifyTokenSet(ctx, "tok-1", id, time.Hour); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	got, err := h.EmailVerifyTokenGet(ctx, "tok-1")
	if err != nil || got != id {
		t.Errorf("Wrong match. expect: %v, got: %v, err: %v", id, got, err)
	}

	// the token expires with its ttl
	s.FastForward(2 * time.Hour)
	if _, err := h.EmailVerifyTokenGet(ctx, "tok-1"); err == nil {
		t.Errorf("Wrong match. expect: expired error, got: nil")
	}

	// a stored value that is not a uuid is rejected
	if err := s.Set("email_verify:tok-bad", "not-a-uuid"); err != nil {
		t.Fatalf("could not seed the key: %v", err)
	}
	if got, err := h.EmailVerifyTokenGet(ctx, "tok-bad"); err == nil || got != uuid.Nil {
		t.Errorf("Wrong match. expect: parse error and nil uuid, got: %v, err: %v", got, err)
	}

	// delete removes the token
	if err := h.EmailVerifyTokenSet(ctx, "tok-2", id, time.Hour); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.EmailVerifyTokenDelete(ctx, "tok-2"); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if s.Exists("email_verify:tok-2") {
		t.Errorf("Wrong match. expect: token deleted")
	}
}

func Test_VerifyLock(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")
	other := uuid.FromStringOrNil("9f6245d5-21b3-4ca7-97ca-069729c87974")

	ok, err := h.VerifyLockAcquire(ctx, id, time.Minute)
	if err != nil || !ok {
		t.Fatalf("Wrong match. expect: acquired, got: %v, err: %v", ok, err)
	}

	ok, err = h.VerifyLockAcquire(ctx, id, time.Minute)
	if err != nil || ok {
		t.Errorf("Wrong match. expect: second acquire denied, got: %v, err: %v", ok, err)
	}

	// the lock is per customer
	ok, err = h.VerifyLockAcquire(ctx, other, time.Minute)
	if err != nil || !ok {
		t.Errorf("Wrong match. expect: other customer acquired, got: %v, err: %v", ok, err)
	}

	if err := h.VerifyLockRelease(ctx, id); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	ok, err = h.VerifyLockAcquire(ctx, id, time.Minute)
	if err != nil || !ok {
		t.Errorf("Wrong match. expect: reacquired after release, got: %v, err: %v", ok, err)
	}
}

func Test_serverDown(t *testing.T) {
	h, s := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("7e6245d5-21b3-4ca7-97ca-069729c87974")

	s.Close()

	if err := h.CustomerSet(ctx, &customer.Customer{ID: id}); err == nil {
		t.Errorf("CustomerSet: expect error")
	}
	if err := h.AccesskeySet(ctx, &accesskey.Accesskey{ID: id}); err == nil {
		t.Errorf("AccesskeySet: expect error")
	}
	if err := h.EmailVerifyTokenSet(ctx, "t", id, time.Minute); err == nil {
		t.Errorf("EmailVerifyTokenSet: expect error")
	}
	if _, err := h.EmailVerifyTokenGet(ctx, "t"); err == nil {
		t.Errorf("EmailVerifyTokenGet: expect error")
	}
	if err := h.EmailVerifyTokenDelete(ctx, "t"); err == nil {
		t.Errorf("EmailVerifyTokenDelete: expect error")
	}
	if ok, err := h.VerifyLockAcquire(ctx, id, time.Minute); err == nil || ok {
		t.Errorf("VerifyLockAcquire: expect error and false, got: %v, %v", ok, err)
	}
	if err := h.VerifyLockRelease(ctx, id); err == nil {
		t.Errorf("VerifyLockRelease: expect error")
	}
	if ok, err := h.ResendCooldownAcquire(ctx, id, time.Minute); err == nil || ok {
		t.Errorf("ResendCooldownAcquire: expect error and false, got: %v, %v", ok, err)
	}
}

func Test_NewHandler_Connect(t *testing.T) {
	_, s := newTestHandler(t)

	h := NewHandler(s.Addr(), "", 0)
	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	s.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after server shutdown, got: nil")
	}
}
