package dbhandler

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

func Test_AI_EndpointColumns(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()
	id := uuid.FromStringOrNil("2f1c0b3e-0a4d-11ef-9a41-8b2d7c5e1111")
	enc := []byte{0x00, 0x01, 0xff, 0x00, 0x7f}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	get := func() *ai.AI {
		mockCache.EXPECT().AIGet(ctx, id).Return(nil, fmt.Errorf(""))
		mockCache.EXPECT().AISet(ctx, gomock.Any())
		res, err := h.AIGet(ctx, id)
		if err != nil {
			t.Fatalf("could not get. err: %v", err)
		}
		return res
	}

	// create with both columns
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AICreate(ctx, &ai.AI{
		Identity:         identity.Identity{ID: id},
		EngineBaseURL:    "https://llm.example.com/v1",
		EngineBaseKeyEnc: enc,
		EngineBaseKey:    "plaintext-must-not-be-stored",
	}); err != nil {
		t.Fatalf("could not create. err: %v", err)
	}
	res := get()
	if res.EngineBaseURL != "https://llm.example.com/v1" || !bytes.Equal(res.EngineBaseKeyEnc, enc) {
		t.Fatalf("the columns did not round trip. got: %q %v", res.EngineBaseURL, res.EngineBaseKeyEnc)
	}
	if res.EngineBaseKey != "" {
		t.Fatalf("the plaintext is db:\"-\" and must never be read or written")
	}

	// update url and key in one statement
	enc2 := []byte("second-ciphertext")
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AIUpdate(ctx, id, map[ai.Field]any{
		ai.FieldEngineBaseURL:    "https://llm-b.example.com",
		ai.FieldEngineBaseKeyEnc: enc2,
	}); err != nil {
		t.Fatalf("could not update. err: %v", err)
	}
	if res = get(); res.EngineBaseURL != "https://llm-b.example.com" || !bytes.Equal(res.EngineBaseKeyEnc, enc2) {
		t.Fatalf("the update did not apply")
	}

	// a nil []byte is NULL
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AIUpdate(ctx, id, map[ai.Field]any{ai.FieldEngineBaseKeyEnc: []byte(nil)}); err != nil {
		t.Fatalf("could not update. err: %v", err)
	}
	if res = get(); res.EngineBaseKeyEnc != nil || res.EngineBaseURL != "https://llm-b.example.com" {
		t.Fatalf("expect NULL ciphertext and an untouched url. got: %v %q", res.EngineBaseKeyEnc, res.EngineBaseURL)
	}
}

// Test_AIDelete_ClearsEndpointKey: the ciphertext must not outlive the AI; the URL is kept.
func Test_AIDelete_ClearsEndpointKey(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()
	id := uuid.FromStringOrNil("2f1c0b3e-0a4d-11ef-9a41-8b2d7c5e2222")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AICreate(ctx, &ai.AI{
		Identity:         identity.Identity{ID: id},
		EngineBaseURL:    "https://llm.example.com/v1",
		EngineBaseKeyEnc: []byte("ciphertext-to-clear"),
	}); err != nil {
		t.Fatalf("could not create. err: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	if err := h.AIDelete(ctx, id); err != nil {
		t.Fatalf("could not delete. err: %v", err)
	}

	mockCache.EXPECT().AIGet(ctx, id).Return(nil, fmt.Errorf(""))
	mockCache.EXPECT().AISet(ctx, gomock.Any())
	res, err := h.AIGet(ctx, id)
	if err != nil {
		t.Fatalf("could not get. err: %v", err)
	}
	if res.TMDelete == nil {
		t.Fatalf("the ai must be soft deleted")
	}
	if res.EngineBaseKeyEnc != nil {
		t.Errorf("the ciphertext must be cleared on delete. got: %v", res.EngineBaseKeyEnc)
	}
	if res.EngineBaseURL != "https://llm.example.com/v1" {
		t.Errorf("the url is kept on delete. got: %q", res.EngineBaseURL)
	}
}

// Test_AI_CacheKeepsCiphertextOnly: whatever the caller put into the plaintext field, the cached
// object carries the ciphertext and never the plaintext.
func Test_AI_CacheKeepsCiphertextOnly(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2023, 1, 3, 21, 35, 2, 809000000, time.UTC); return &t }()
	id := uuid.FromStringOrNil("2f1c0b3e-0a4d-11ef-9a41-8b2d7c5e3333")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	var cached *ai.AI
	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().AISet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, a *ai.AI) error {
		cached = a
		return nil
	})
	if err := h.AICreate(ctx, &ai.AI{
		Identity:         identity.Identity{ID: id},
		EngineBaseURL:    "https://llm.example.com",
		EngineBaseKeyEnc: []byte("cipher"),
		EngineBaseKey:    "plaintext-sentinel",
	}); err != nil {
		t.Fatalf("could not create. err: %v", err)
	}

	if cached == nil || cached.EngineBaseKey != "" || !bytes.Equal(cached.EngineBaseKeyEnc, []byte("cipher")) {
		t.Fatalf("the cache must hold the ciphertext only. got: %+v", cached)
	}
}
