package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-message-manager/models/message"
)

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatalf("NewHandler() did not return *handler")
	}
	t.Cleanup(func() {
		_ = h.Cache.Close()
	})

	return h, mr
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

func Test_MessageSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	m := &message.Message{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("66666666-7777-8888-9999-000000000000"),
		},
		Type:         message.TypeSMS,
		ProviderName: message.ProviderNameTelnyx,
		Text:         "cached text",
		Direction:    message.DirectionOutbound,
	}

	if err := h.MessageSet(ctx, m); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "message:" + id.String()
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. expect the key %s to exist", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong ttl. expect: %v, got: %v", 24*time.Hour, ttl)
	}

	res, err := h.MessageGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != id || res.CustomerID != m.CustomerID || res.Text != "cached text" ||
		res.ProviderName != message.ProviderNameTelnyx || res.Direction != message.DirectionOutbound {
		t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", m, res)
	}

	// just before the ttl the entry is still served, after it the entry is gone.
	mr.FastForward(24*time.Hour - time.Second)
	if _, err := h.MessageGet(ctx, id); err != nil {
		t.Errorf("Wrong match. expect the entry to be alive before the ttl, got: %v", err)
	}

	mr.FastForward(2 * time.Second)
	if _, err := h.MessageGet(ctx, id); err == nil {
		t.Errorf("Wrong match. expect the entry to expire after the ttl")
	}
}

func Test_MessageGet_errors(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	missing := uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if res, err := h.MessageGet(ctx, missing); err == nil || res != nil {
		t.Errorf("Wrong match for a missing key. expect: error, got: res: %v, err: %v", res, err)
	}

	corrupt := uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")
	if err := mr.Set("message:"+corrupt.String(), "{not json"); err != nil {
		t.Fatalf("could not prepare the corrupt entry: %v", err)
	}
	if res, err := h.MessageGet(ctx, corrupt); err == nil || res != nil {
		t.Errorf("Wrong match for a corrupt entry. expect: error, got: res: %v, err: %v", res, err)
	}
}

func Test_MessageSet_closedServer(t *testing.T) {
	h, mr := newTestHandler(t)
	mr.Close()

	m := &message.Message{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000"),
		},
	}
	if err := h.MessageSet(context.Background(), m); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_delSerialize(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-111111111111")
	if err := h.MessageSet(ctx, &message.Message{Identity: commonidentity.Identity{ID: id}}); err != nil {
		t.Fatalf("could not prepare the entry: %v", err)
	}

	key := "message:" + id.String()
	if err := h.delSerialize(ctx, key); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists(key) {
		t.Errorf("Wrong match. expect the key %s to be deleted", key)
	}

	mr.Close()
	if err := h.delSerialize(ctx, key); err == nil {
		t.Errorf("Wrong match. expect: error against a closed server, got: nil")
	}
}

func Test_setSerialize_unmarshalableValue(t *testing.T) {
	h, mr := newTestHandler(t)

	// a channel cannot be encoded as JSON, so nothing may be written.
	if err := h.setSerialize(context.Background(), "key-unmarshalable", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if mr.Exists("key-unmarshalable") {
		t.Errorf("Wrong match. expect: no key written, got: key exists")
	}
}
