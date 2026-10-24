package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/models/webhook"
)

func newMiniredis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to create miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	return mr
}

func Test_AccountSetGet(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	ctx := context.Background()

	id := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")
	a := &account.Account{
		ID:            id,
		WebhookMethod: webhook.MethodTypePOST,
		WebhookURI:    "https://example.com/hook",
		WebhookSecret: "placeholder-secret",
	}

	if err := h.AccountSet(ctx, a); err != nil {
		t.Fatalf("Could not set the account. err: %v", err)
	}

	// the stored key and expiry are part of the contract with other readers.
	key := "webhook.account:22222222-2222-2222-2222-222222222222"
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect the key to exist: %s", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: 24h, got: %v", ttl)
	}

	res, err := h.AccountGet(ctx, id)
	if err != nil {
		t.Fatalf("Could not get the account. err: %v", err)
	}
	if res.ID != id || res.WebhookMethod != webhook.MethodTypePOST || res.WebhookURI != "https://example.com/hook" || res.WebhookSecret != "placeholder-secret" {
		t.Errorf("Wrong match. got: %+v", res)
	}
}

func Test_AccountGet_miss(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	res, err := h.AccountGet(context.Background(), uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"))
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: %+v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

func Test_AccountGet_corrupted(t *testing.T) {
	h, mr := setupTestHandler(t)
	defer mr.Close()

	id := uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444")
	if err := mr.Set("webhook.account:44444444-4444-4444-4444-444444444444", "{not json"); err != nil {
		t.Fatalf("Could not seed the key. err: %v", err)
	}

	res, err := h.AccountGet(context.Background(), id)
	if err == nil {
		t.Errorf("Wrong match. expect: error, got: %+v", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %+v", res)
	}
}

func Test_AccountSet_server_down(t *testing.T) {
	h, mr := setupTestHandler(t)
	mr.Close()

	a := &account.Account{ID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555")}
	if err := h.AccountSet(context.Background(), a); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_NewHandler_Connect(t *testing.T) {
	mr := newMiniredis(t)

	h := NewHandler(mr.Addr(), "", 0)
	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server stopped, got: nil")
	}
}

// Test_NewHandler_passes_credentials_and_db checks that the password and the
// database index reach the redis client, not just the handler struct.
func Test_NewHandler_passes_credentials_and_db(t *testing.T) {
	mr := newMiniredis(t)
	mr.RequireAuth("placeholder")

	if err := NewHandler(mr.Addr(), "placeholder-other", 0).Connect(); err == nil {
		t.Errorf("Wrong match. expect: error with a wrong password, got: nil")
	}

	h := NewHandler(mr.Addr(), "placeholder", 2)
	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok with the right password, got: %v", err)
	}

	id := uuid.FromStringOrNil("88888888-8888-8888-8888-888888888888")
	if err := h.AccountSet(context.Background(), &account.Account{ID: id}); err != nil {
		t.Fatalf("Could not set the account. err: %v", err)
	}

	key := "webhook.account:88888888-8888-8888-8888-888888888888"
	mr.Select(2)
	if !mr.Exists(key) {
		t.Errorf("Wrong match. expect the key in database 2")
	}
	mr.Select(0)
	if mr.Exists(key) {
		t.Errorf("Wrong match. expect no key in database 0")
	}
}
