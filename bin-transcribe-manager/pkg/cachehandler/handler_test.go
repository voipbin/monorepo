package cachehandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

func newTestCacheHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start the miniredis. err: %v", err)
	}
	t.Cleanup(mr.Close)

	h := &handler{
		Addr:  mr.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr()}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func Test_TranscribeSetGet(t *testing.T) {
	id := uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000001")

	tests := []struct {
		name string

		transcribe *transcribe.Transcribe
		expectKey  string
	}{
		{
			name: "normal",
			transcribe: &transcribe.Transcribe{
				Identity:      commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000002")},
				ReferenceType: transcribe.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000003"),
				Status:        transcribe.StatusProgressing,
				Language:      "en-US",
				StreamingIDs:  []uuid.UUID{uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000004")},
			},
			expectKey: "transcribe:transcribe:7a0e5a40-0000-11ee-8c10-000000000001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestCacheHandler(t)
			ctx := context.Background()

			if errSet := h.TranscribeSet(ctx, tt.transcribe); errSet != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
			}

			if !mr.Exists(tt.expectKey) {
				t.Errorf("Wrong match. expect key: %s, got: not found", tt.expectKey)
			}
			if ttl := mr.TTL(tt.expectKey); ttl != 24*time.Hour {
				t.Errorf("Wrong match. expect ttl: %v, got: %v", 24*time.Hour, ttl)
			}

			res, err := h.TranscribeGet(ctx, tt.transcribe.ID)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.ID != tt.transcribe.ID || res.ReferenceType != tt.transcribe.ReferenceType || res.Language != tt.transcribe.Language || len(res.StreamingIDs) != 1 || res.StreamingIDs[0] != tt.transcribe.StreamingIDs[0] {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.transcribe, res)
			}
		})
	}
}

func Test_TranscriptSetGet(t *testing.T) {
	id := uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000011")

	h, mr := newTestCacheHandler(t)
	ctx := context.Background()

	tr := &transcript.Transcript{
		Identity:     commonidentity.Identity{ID: id},
		TranscribeID: uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000012"),
		Direction:    transcript.DirectionIn,
		Message:      "hello world",
		OffsetMs:     1500,
	}

	if errSet := h.TranscriptSet(ctx, tr); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}

	key := "transcribe:transcript:7a0e5a40-0000-11ee-8c10-000000000011"
	if !mr.Exists(key) {
		t.Errorf("Wrong match. expect key: %s, got: not found", key)
	}

	res, err := h.TranscriptGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Message != "hello world" || res.OffsetMs != 1500 || res.Direction != transcript.DirectionIn || res.TranscribeID != tr.TranscribeID {
		t.Errorf("Wrong match. expect: %v, got: %v", tr, res)
	}
}

func Test_StreamingSetGet(t *testing.T) {
	id := uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000021")

	h, mr := newTestCacheHandler(t)
	ctx := context.Background()

	st := &streaming.Streaming{
		Identity:     commonidentity.Identity{ID: id},
		TranscribeID: uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000022"),
		Language:     "ko-KR",
		Direction:    transcript.DirectionOut,
	}

	if errSet := h.StreamingSet(ctx, st); errSet != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errSet)
	}

	key := "transcribe:streaming:7a0e5a40-0000-11ee-8c10-000000000021"
	if !mr.Exists(key) {
		t.Errorf("Wrong match. expect key: %s, got: not found", key)
	}

	res, err := h.StreamingGet(ctx, id)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.Language != "ko-KR" || res.Direction != transcript.DirectionOut || res.TranscribeID != st.TranscribeID {
		t.Errorf("Wrong match. expect: %v, got: %v", st, res)
	}
}

func Test_Get_notFoundAndBrokenJSON(t *testing.T) {
	id := uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000031")

	tests := []struct {
		name string

		// raw value stored under the key before the call. empty means no key.
		rawValue string
		key      string

		get func(h *handler) error

		expectNotFound bool
	}{
		{
			name: "transcribe missing",
			key:  "transcribe:transcribe:" + id.String(),
			get: func(h *handler) error {
				_, err := h.TranscribeGet(context.Background(), id)
				return err
			},
			expectNotFound: true,
		},
		{
			name: "transcript missing",
			key:  "transcribe:transcript:" + id.String(),
			get: func(h *handler) error {
				_, err := h.TranscriptGet(context.Background(), id)
				return err
			},
			expectNotFound: true,
		},
		{
			name: "streaming missing",
			key:  "transcribe:streaming:" + id.String(),
			get: func(h *handler) error {
				_, err := h.StreamingGet(context.Background(), id)
				return err
			},
			expectNotFound: true,
		},
		{
			name:     "transcribe broken json",
			key:      "transcribe:transcribe:" + id.String(),
			rawValue: "{not-json",
			get: func(h *handler) error {
				_, err := h.TranscribeGet(context.Background(), id)
				return err
			},
		},
		{
			name:     "transcript broken json",
			key:      "transcribe:transcript:" + id.String(),
			rawValue: "{not-json",
			get: func(h *handler) error {
				_, err := h.TranscriptGet(context.Background(), id)
				return err
			},
		},
		{
			name:     "streaming broken json",
			key:      "transcribe:streaming:" + id.String(),
			rawValue: "{not-json",
			get: func(h *handler) error {
				_, err := h.StreamingGet(context.Background(), id)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestCacheHandler(t)

			if tt.rawValue != "" {
				if errSet := mr.Set(tt.key, tt.rawValue); errSet != nil {
					t.Fatalf("could not seed the key. err: %v", errSet)
				}
			}

			err := tt.get(h)
			if err == nil {
				t.Fatalf("Wrong match. expect: err, got: ok")
			}

			if gotNotFound := errors.Is(err, redis.Nil); gotNotFound != tt.expectNotFound {
				t.Errorf("Wrong match. expect not found: %v, got: %v (err: %v)", tt.expectNotFound, gotNotFound, err)
			}
		})
	}
}

func Test_Set_serverDown(t *testing.T) {
	h, mr := newTestCacheHandler(t)
	mr.Close()

	ctx := context.Background()
	id := uuid.FromStringOrNil("7a0e5a40-0000-11ee-8c10-000000000041")

	if err := h.TranscribeSet(ctx, &transcribe.Transcribe{Identity: commonidentity.Identity{ID: id}}); err == nil {
		t.Errorf("Wrong match. TranscribeSet expect: err, got: ok")
	}
	if err := h.TranscriptSet(ctx, &transcript.Transcript{Identity: commonidentity.Identity{ID: id}}); err == nil {
		t.Errorf("Wrong match. TranscriptSet expect: err, got: ok")
	}
	if err := h.StreamingSet(ctx, &streaming.Streaming{Identity: commonidentity.Identity{ID: id}}); err == nil {
		t.Errorf("Wrong match. StreamingSet expect: err, got: ok")
	}
}

func Test_delSerialize(t *testing.T) {
	h, mr := newTestCacheHandler(t)
	ctx := context.Background()

	key := "transcribe:transcribe:7a0e5a40-0000-11ee-8c10-000000000051"
	if errSet := mr.Set(key, "{}"); errSet != nil {
		t.Fatalf("could not seed the key. err: %v", errSet)
	}

	if err := h.delSerialize(ctx, key); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists(key) {
		t.Errorf("Wrong match. expect key removed, got: still exists")
	}

	// deleting a missing key is not an error
	if err := h.delSerialize(ctx, key); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mr.Close()
	if err := h.delSerialize(ctx, key); err == nil {
		t.Errorf("Wrong match. expect: err on a closed server, got: ok")
	}
}
