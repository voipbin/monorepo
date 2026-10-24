package cachehandler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/alicebob/miniredis/v2"
	"github.com/alicebob/miniredis/v2/server"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
)

const (
	testTransferID       = "2d8b1f3a-0001-11ee-9a11-000000000001"
	testTransfererCallID = "2d8b1f3a-0002-11ee-9a11-000000000002"
	testGroupcallID      = "2d8b1f3a-0003-11ee-9a11-000000000003"
	testConfbridgeID     = "2d8b1f3a-0004-11ee-9a11-000000000004"
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

func newTestTransfer() *transfer.Transfer {
	return &transfer.Transfer{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil(testTransferID),
		},
		Type:             transfer.TypeAttended,
		TransfererCallID: uuid.FromStringOrNil(testTransfererCallID),
		GroupcallID:      uuid.FromStringOrNil(testGroupcallID),
		ConfbridgeID:     uuid.FromStringOrNil(testConfbridgeID),
	}
}

func Test_NewHandler_Connect(t *testing.T) {
	mr := miniredis.RunT(t)

	h := NewHandler(mr.Addr(), "", 0)
	if h == nil {
		t.Fatalf("Wrong match. expect: handler, got: nil")
	}

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error after the server is closed, got: ok")
	}
}

func Test_TransferSet_writesThreeKeys(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	tr := newTestTransfer()
	if err := h.TransferSet(ctx, tr); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	keys := []string{
		"transfer:transfer:" + testTransferID,
		"transfer:transferer_call_id:" + testTransfererCallID,
		"transfer:groupcall_id:" + testGroupcallID,
	}

	var first string
	for i, key := range keys {
		got, err := mr.Get(key)
		if err != nil {
			t.Fatalf("Wrong match. key: %s, expect: stored, got: %v", key, err)
		}
		if i == 0 {
			first = got
		} else if got != first {
			t.Errorf("Wrong match. key: %s, expect: same body as the first key, got: %s", key, got)
		}

		if ttl := mr.TTL(key); ttl != 24*time.Hour {
			t.Errorf("Wrong match. key: %s, expect ttl: %v, got: %v", key, 24*time.Hour, ttl)
		}
	}
}

func Test_TransferGet_restoresFields(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	// the body is written as a literal JSON so that the getters are verified
	// independently of the setter.
	body := `{"id":"` + testTransferID + `","type":"blind","transferer_call_id":"` + testTransfererCallID + `","transferee_call_id":"2d8b1f3a-0005-11ee-9a11-000000000005","groupcall_id":"` + testGroupcallID + `","confbridge_id":"` + testConfbridgeID + `"}`
	if err := mr.Set("transfer:transfer:"+testTransferID, body); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}
	if err := mr.Set("transfer:transferer_call_id:"+testTransfererCallID, body); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}
	if err := mr.Set("transfer:groupcall_id:"+testGroupcallID, body); err != nil {
		t.Fatalf("could not seed. err: %v", err)
	}

	getters := []struct {
		name string
		get  func() (*transfer.Transfer, error)
	}{
		{"by id", func() (*transfer.Transfer, error) {
			return h.TransferGet(ctx, uuid.FromStringOrNil(testTransferID))
		}},
		{"by transferer call id", func() (*transfer.Transfer, error) {
			return h.TransferGetByTransfererCallID(ctx, uuid.FromStringOrNil(testTransfererCallID))
		}},
		{"by groupcall id", func() (*transfer.Transfer, error) {
			return h.TransferGetByGroupcallID(ctx, uuid.FromStringOrNil(testGroupcallID))
		}},
	}

	for _, g := range getters {
		t.Run(g.name, func(t *testing.T) {
			res, err := g.get()
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if res.ID.String() != testTransferID {
				t.Errorf("Wrong id. expect: %s, got: %s", testTransferID, res.ID)
			}
			if res.Type != transfer.TypeBlind {
				t.Errorf("Wrong type. expect: %s, got: %s", transfer.TypeBlind, res.Type)
			}
			if res.TransfererCallID.String() != testTransfererCallID {
				t.Errorf("Wrong transferer call id. expect: %s, got: %s", testTransfererCallID, res.TransfererCallID)
			}
			if res.TransfereeCallID.String() != "2d8b1f3a-0005-11ee-9a11-000000000005" {
				t.Errorf("Wrong transferee call id. got: %s", res.TransfereeCallID)
			}
			if res.GroupcallID.String() != testGroupcallID {
				t.Errorf("Wrong groupcall id. expect: %s, got: %s", testGroupcallID, res.GroupcallID)
			}
			if res.ConfbridgeID.String() != testConfbridgeID {
				t.Errorf("Wrong confbridge id. expect: %s, got: %s", testConfbridgeID, res.ConfbridgeID)
			}
		})
	}
}

func Test_TransferGet_errors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil(testTransferID)

	t.Run("missing key returns redis.Nil", func(t *testing.T) {
		h, _ := newTestHandler(t)

		if _, err := h.TransferGet(ctx, id); !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
		}
		if _, err := h.TransferGetByTransfererCallID(ctx, id); !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
		}
		if _, err := h.TransferGetByGroupcallID(ctx, id); !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect: redis.Nil, got: %v", err)
		}
	})

	t.Run("corrupted body returns an unmarshal error", func(t *testing.T) {
		h, mr := newTestHandler(t)
		if err := mr.Set("transfer:transfer:"+testTransferID, "{not-json"); err != nil {
			t.Fatalf("could not seed. err: %v", err)
		}

		res, err := h.TransferGet(ctx, id)
		if err == nil || errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect: unmarshal error, got: %v", err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})

	t.Run("closed server returns an error", func(t *testing.T) {
		h, mr := newTestHandler(t)
		mr.Close()

		ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		if _, err := h.TransferGet(ctx, id); err == nil {
			t.Errorf("Wrong match. expect: error, got: ok")
		}
		if err := h.TransferSet(ctx, newTestTransfer()); err == nil {
			t.Errorf("Wrong match. expect: error, got: ok")
		}
	})
}

func Test_setSerialize_marshalError(t *testing.T) {
	h, _ := newTestHandler(t)

	// a channel can not be marshaled to json.
	if err := h.setSerialize(context.Background(), "key", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: marshal error, got: ok")
	}
}

// Test_TransferSet_failsOnNthSet makes the nth SET fail and verifies that the error is
// propagated and that the later keys are not written.
func Test_TransferSet_failsOnNthSet(t *testing.T) {
	tests := []struct {
		name string
		nth  int32

		expectKeys   []string
		unexpectKeys []string
	}{
		{
			name: "first set fails",
			nth:  1,

			unexpectKeys: []string{
				"transfer:transfer:" + testTransferID,
				"transfer:transferer_call_id:" + testTransfererCallID,
				"transfer:groupcall_id:" + testGroupcallID,
			},
		},
		{
			name: "second set fails",
			nth:  2,

			expectKeys: []string{"transfer:transfer:" + testTransferID},
			unexpectKeys: []string{
				"transfer:transferer_call_id:" + testTransfererCallID,
				"transfer:groupcall_id:" + testGroupcallID,
			},
		},
		{
			name: "third set fails",
			nth:  3,

			expectKeys: []string{
				"transfer:transfer:" + testTransferID,
				"transfer:transferer_call_id:" + testTransfererCallID,
			},
			unexpectKeys: []string{"transfer:groupcall_id:" + testGroupcallID},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)

			var count int32
			srv := mr.Server()
			srv.SetPreHook(func(c *server.Peer, cmd string, args ...string) bool {
				if cmd != "SET" {
					return false
				}
				if atomic.AddInt32(&count, 1) == tt.nth {
					c.WriteError("ERR injected failure")
					return true
				}
				return false
			})
			t.Cleanup(func() { srv.SetPreHook(nil) })

			if err := h.TransferSet(context.Background(), newTestTransfer()); err == nil {
				t.Fatalf("Wrong match. expect: error, got: ok")
			}

			for _, key := range tt.expectKeys {
				if !mr.Exists(key) {
					t.Errorf("Wrong match. key: %s, expect: written, got: missing", key)
				}
			}
			for _, key := range tt.unexpectKeys {
				if mr.Exists(key) {
					t.Errorf("Wrong match. key: %s, expect: not written, got: exists", key)
				}
			}
		})
	}
}
