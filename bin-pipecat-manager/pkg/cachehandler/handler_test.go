package cachehandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

// newTestHandler returns a handler that is connected to a private in-memory redis server.
func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatalf("Wrong type. expect: *handler")
	}

	return h, mr
}

func Test_NewHandler(t *testing.T) {
	mr := miniredis.RunT(t)

	// the password is set in a separate statement on purpose
	password := "fake-cache-auth"
	database := 3

	h, ok := NewHandler(mr.Addr(), password, database).(*handler)
	if !ok {
		t.Fatalf("Wrong type. expect: *handler")
	}

	if h.Addr != mr.Addr() || h.Password != password || h.DB != database {
		t.Errorf("Wrong settings. addr: %s, db: %d", h.Addr, h.DB)
	}
	if h.Cache == nil || h.Locker == nil || h.mapMutex == nil {
		t.Errorf("Wrong handler. cache: %v, locker: %v, mapMutex: %v", h.Cache, h.Locker, h.mapMutex)
	}

	opts := h.Cache.Options()
	if opts.Addr != mr.Addr() || opts.Password != password || opts.DB != database {
		t.Errorf("Wrong client options. addr: %s, db: %d", opts.Addr, opts.DB)
	}
}

func Test_Connect(t *testing.T) {
	t.Run("connected", func(t *testing.T) {
		h, _ := newTestHandler(t)

		if err := h.Connect(); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("server is down", func(t *testing.T) {
		h, mr := newTestHandler(t)
		mr.Close()

		if err := h.Connect(); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})
}

func Test_PipecatcallSetGet(t *testing.T) {
	id := uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0001")
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall
	}{
		{
			name: "full pipecatcall",
			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         id,
					CustomerID: uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0002"),
				},
				ActiveflowID:  uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0003"),
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0004"),
				HostID:        "host-a",
				LLMType:       pipecatcall.LLMType("openai.test-model"),
				LLMMessages:   []map[string]any{{"role": "system", "content": "hello"}},
				STTType:       pipecatcall.STTTypeDeepgram,
				STTLanguage:   "en",
				TTSType:       pipecatcall.TTSTypeCartesia,
				TTSLanguage:   "en",
				TTSVoiceID:    "voice-a",
				TMCreate:      &createdAt,
			},
		},
		{
			name: "minimal pipecatcall",
			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{ID: id},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)
			ctx := context.Background()

			if err := h.PipecatcallSet(ctx, tt.pc); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			// the value is stored under the pipecatcall key and expires in 24 hours
			key := "pipecatcall:" + tt.pc.ID.String()
			if !mr.Exists(key) {
				t.Fatalf("Wrong key. expect: %s, keys: %v", key, mr.Keys())
			}
			if ttl := mr.TTL(key); ttl != 24*time.Hour {
				t.Errorf("Wrong ttl. expect: %v, got: %v", 24*time.Hour, ttl)
			}

			res, err := h.PipecatcallGet(ctx, tt.pc.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.pc, res) {
				t.Errorf("Wrong match.\nexpect: %+v\ngot: %+v", tt.pc, res)
			}
		})
	}
}

func Test_PipecatcallGet_error(t *testing.T) {
	id := uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0010")

	t.Run("not cached", func(t *testing.T) {
		h, _ := newTestHandler(t)

		res, err := h.PipecatcallGet(context.Background(), id)
		if err != redis.Nil {
			t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})

	t.Run("broken json", func(t *testing.T) {
		h, mr := newTestHandler(t)

		if err := mr.Set("pipecatcall:"+id.String(), "{not-json"); err != nil {
			t.Fatalf("could not seed the value. err: %v", err)
		}

		res, err := h.PipecatcallGet(context.Background(), id)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", res)
		}
	})
}

func Test_PipecatcallSet_error(t *testing.T) {
	h, mr := newTestHandler(t)

	// an already canceled context fails before any command reaches the server
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	pc := &pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0020")},
	}
	if err := h.PipecatcallSet(ctx, pc); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}

	if len(mr.Keys()) != 0 {
		t.Errorf("Wrong keys. expect: none, got: %v", mr.Keys())
	}
}
