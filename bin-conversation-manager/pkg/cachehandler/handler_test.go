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

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
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

func Test_cache_roundtrip(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	idAccount := uuid.FromStringOrNil("11111111-0000-4000-8000-000000000001")
	idConversation := uuid.FromStringOrNil("22222222-0000-4000-8000-000000000002")
	idMessage := uuid.FromStringOrNil("33333333-0000-4000-8000-000000000003")
	idMedia := uuid.FromStringOrNil("44444444-0000-4000-8000-000000000004")
	customerID := uuid.FromStringOrNil("55555555-0000-4000-8000-000000000005")

	tests := []struct {
		name string

		expectKey string
		set       func(h *handler) error
		get       func(h *handler) (any, error)
		expectRes any
	}{
		{
			name:      "account",
			expectKey: "conversation.account:" + idAccount.String(),
			set: func(h *handler) error {
				return h.AccountSet(context.Background(), &account.Account{
					Identity: commonidentity.Identity{ID: idAccount, CustomerID: customerID},
					Type:     account.TypeLine,
					Name:     "test account",
					TMCreate: &ts,
				})
			},
			get: func(h *handler) (any, error) { return h.AccountGet(context.Background(), idAccount) },
			expectRes: &account.Account{
				Identity: commonidentity.Identity{ID: idAccount, CustomerID: customerID},
				Type:     account.TypeLine,
				Name:     "test account",
				TMCreate: &ts,
			},
		},
		{
			name:      "conversation",
			expectKey: "conversation.conversation:" + idConversation.String(),
			set: func(h *handler) error {
				return h.ConversationSet(context.Background(), &conversation.Conversation{
					Identity:  commonidentity.Identity{ID: idConversation, CustomerID: customerID},
					AccountID: idAccount,
					Type:      conversation.TypeLine,
					DialogID:  "dialog-1",
					TMCreate:  &ts,
				})
			},
			get: func(h *handler) (any, error) { return h.ConversationGet(context.Background(), idConversation) },
			expectRes: &conversation.Conversation{
				Identity:  commonidentity.Identity{ID: idConversation, CustomerID: customerID},
				AccountID: idAccount,
				Type:      conversation.TypeLine,
				DialogID:  "dialog-1",
				TMCreate:  &ts,
			},
		},
		{
			name:      "message",
			expectKey: "conversation.message:" + idMessage.String(),
			set: func(h *handler) error {
				return h.MessageSet(context.Background(), &message.Message{
					Identity:       commonidentity.Identity{ID: idMessage, CustomerID: customerID},
					ConversationID: idConversation,
					Direction:      message.DirectionIncoming,
					Text:           "hello",
					TMCreate:       &ts,
				})
			},
			get: func(h *handler) (any, error) { return h.MessageGet(context.Background(), idMessage) },
			expectRes: &message.Message{
				Identity:       commonidentity.Identity{ID: idMessage, CustomerID: customerID},
				ConversationID: idConversation,
				Direction:      message.DirectionIncoming,
				Text:           "hello",
				TMCreate:       &ts,
			},
		},
		{
			name:      "media",
			expectKey: "conversation.media:" + idMedia.String(),
			set: func(h *handler) error {
				return h.MediaSet(context.Background(), &media.Media{
					Identity: commonidentity.Identity{ID: idMedia, CustomerID: customerID},
					Type:     media.TypeImage,
					Filename: "a.png",
					TMCreate: &ts,
				})
			},
			get: func(h *handler) (any, error) { return h.MediaGet(context.Background(), idMedia) },
			expectRes: &media.Media{
				Identity: commonidentity.Identity{ID: idMedia, CustomerID: customerID},
				Type:     media.TypeImage,
				Filename: "a.png",
				TMCreate: &ts,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestCache(t)

			if errSet := tt.set(h); errSet != nil {
				t.Fatalf("Set() error = %v", errSet)
			}

			// the entry lives under a literal key for one day
			if !mr.Exists(tt.expectKey) {
				t.Fatalf("key %q was not stored, keys: %v", tt.expectKey, mr.Keys())
			}
			if got := mr.TTL(tt.expectKey); got != 24*time.Hour {
				t.Errorf("TTL of %q = %v, want %v", tt.expectKey, got, 24*time.Hour)
			}

			res, err := tt.get(h)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Get() = %+v, want %+v", res, tt.expectRes)
			}

			// after the TTL passes the entry is gone
			mr.FastForward(24 * time.Hour)
			if _, errExpired := tt.get(h); errExpired != redis.Nil {
				t.Errorf("Get() after expiry error = %v, want %v", errExpired, redis.Nil)
			}
		})
	}
}

func Test_cache_get_missing(t *testing.T) {
	h, _ := newTestCache(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("66666666-0000-4000-8000-000000000006")

	if res, err := h.AccountGet(ctx, id); err != redis.Nil || res != nil {
		t.Errorf("AccountGet() = %v, %v, want nil, %v", res, err, redis.Nil)
	}
	if res, err := h.ConversationGet(ctx, id); err != redis.Nil || res != nil {
		t.Errorf("ConversationGet() = %v, %v, want nil, %v", res, err, redis.Nil)
	}
	if res, err := h.MessageGet(ctx, id); err != redis.Nil || res != nil {
		t.Errorf("MessageGet() = %v, %v, want nil, %v", res, err, redis.Nil)
	}
	if res, err := h.MediaGet(ctx, id); err != redis.Nil || res != nil {
		t.Errorf("MediaGet() = %v, %v, want nil, %v", res, err, redis.Nil)
	}
}

func Test_cache_get_stored_value(t *testing.T) {
	id := uuid.FromStringOrNil("77777777-0000-4000-8000-000000000007")
	key := "conversation.account:" + id.String()

	t.Run("a value that is not JSON is reported as an unmarshal error", func(t *testing.T) {
		h, mr := newTestCache(t)
		if err := mr.Set(key, "{not json"); err != nil {
			t.Fatalf("could not seed the cache: %v", err)
		}

		res, err := h.AccountGet(context.Background(), id)
		if err == nil || err == redis.Nil {
			t.Errorf("AccountGet() error = %v, want an unmarshal error", err)
		}
		if res != nil {
			t.Errorf("AccountGet() = %v, want nil", res)
		}
	})

	t.Run("a JSON null yields an empty value without an error", func(t *testing.T) {
		// characterization of the current behavior: null is accepted and leaves the zero value
		h, mr := newTestCache(t)
		if err := mr.Set(key, "null"); err != nil {
			t.Fatalf("could not seed the cache: %v", err)
		}

		res, err := h.AccountGet(context.Background(), id)
		if err != nil {
			t.Fatalf("AccountGet() error = %v", err)
		}
		if res == nil || res.ID != uuid.Nil {
			t.Errorf("AccountGet() = %+v, want the zero value", res)
		}
	})

	t.Run("a value of the wrong JSON type is reported as an error", func(t *testing.T) {
		h, mr := newTestCache(t)
		if err := mr.Set(key, `"text"`); err != nil {
			t.Fatalf("could not seed the cache: %v", err)
		}

		if res, err := h.AccountGet(context.Background(), id); err == nil || res != nil {
			t.Errorf("AccountGet() = %v, %v, want nil and an error", res, err)
		}
	})
}

func Test_cache_set_server_down(t *testing.T) {
	h, mr := newTestCache(t)
	mr.Close()

	ctx := context.Background()
	if err := h.AccountSet(ctx, &account.Account{}); err == nil {
		t.Errorf("AccountSet() error = nil, want an error")
	}
	if err := h.ConversationSet(ctx, &conversation.Conversation{}); err == nil {
		t.Errorf("ConversationSet() error = nil, want an error")
	}
	if err := h.MessageSet(ctx, &message.Message{}); err == nil {
		t.Errorf("MessageSet() error = nil, want an error")
	}
	if err := h.MediaSet(ctx, &media.Media{}); err == nil {
		t.Errorf("MediaSet() error = nil, want an error")
	}
}

func Test_Connect(t *testing.T) {
	t.Run("succeeds against a running server", func(t *testing.T) {
		h, _ := newTestCache(t)
		if err := h.Connect(); err != nil {
			t.Errorf("Connect() error = %v", err)
		}
	})

	t.Run("fails after the server stopped", func(t *testing.T) {
		h, mr := newTestCache(t)
		mr.Close()
		if err := h.Connect(); err == nil {
			t.Errorf("Connect() error = nil, want an error")
		}
	})
}

func Test_NewHandler(t *testing.T) {
	t.Run("the password is sent to the server", func(t *testing.T) {
		mr := miniredis.RunT(t)
		mr.RequireAuth(strings.Repeat("r", 12))

		bad := NewHandler(mr.Addr(), strings.Repeat("w", 12), 0)
		if err := bad.Connect(); err == nil {
			t.Errorf("Connect() with a wrong password error = nil, want an error")
		}

		good := NewHandler(mr.Addr(), strings.Repeat("r", 12), 0)
		if err := good.Connect(); err != nil {
			t.Errorf("Connect() with the right password error = %v", err)
		}
	})

	t.Run("the database index is selected", func(t *testing.T) {
		mr := miniredis.RunT(t)
		h := NewHandler(mr.Addr(), "", 3)

		id := uuid.FromStringOrNil("88888888-0000-4000-8000-000000000008")
		if err := h.MediaSet(context.Background(), &media.Media{Identity: commonidentity.Identity{ID: id}}); err != nil {
			t.Fatalf("MediaSet() error = %v", err)
		}

		key := "conversation.media:" + id.String()
		mr.Select(3)
		if !mr.Exists(key) {
			t.Errorf("key %q is not stored in database 3", key)
		}
		mr.Select(0)
		if mr.Exists(key) {
			t.Errorf("key %q must not be stored in database 0", key)
		}
	})
}
