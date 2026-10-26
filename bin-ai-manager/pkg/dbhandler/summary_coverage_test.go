package dbhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/cachehandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_SummaryDelete(t *testing.T) {
	resetTestDB(t)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := &handler{utilHandler: mockUtil, db: dbTest, cache: mockCache}
	ctx := context.Background()

	id := uuid.FromStringOrNil("e5000000-0000-0000-0000-000000000001")
	createTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	deleteTime := time.Date(2024, 1, 2, 3, 6, 5, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&createTime)
	mockCache.EXPECT().SummarySet(ctx, gomock.Any()).Return(nil)
	if err := h.SummaryCreate(ctx, &summary.Summary{
		Identity:      identity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")},
		ReferenceType: summary.ReferenceTypeTranscribe,
		ReferenceID:   uuid.FromStringOrNil("e5000000-0000-0000-0000-0000000000a1"),
		Status:        summary.StatusDone,
		Language:      "en-US",
		Content:       "text",
	}); err != nil {
		t.Fatalf("create failed. err: %v", err)
	}

	// an unrelated summary that must be left untouched by the delete.
	bystanderID := uuid.FromStringOrNil("e5000000-0000-0000-0000-000000000002")
	mockUtil.EXPECT().TimeNow().Return(&createTime)
	mockCache.EXPECT().SummarySet(ctx, gomock.Any()).Return(nil)
	if err := h.SummaryCreate(ctx, &summary.Summary{
		Identity:      identity.Identity{ID: bystanderID, CustomerID: uuid.FromStringOrNil("c1000000-0000-0000-0000-000000000001")},
		ReferenceType: summary.ReferenceTypeTranscribe,
		ReferenceID:   uuid.FromStringOrNil("e5000000-0000-0000-0000-0000000000a2"),
		Status:        summary.StatusDone,
		Language:      "en-US",
		Content:       "other",
	}); err != nil {
		t.Fatalf("bystander create failed. err: %v", err)
	}

	// delete stamps tm_update and tm_delete with the same time and refreshes
	// the cache from the stored row.
	mockUtil.EXPECT().TimeNow().Return(&deleteTime)
	mockCache.EXPECT().SummarySet(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, got *summary.Summary) error {
		if got.ID != id || got.TMDelete == nil || !got.TMDelete.Equal(deleteTime) || got.TMUpdate == nil || !got.TMUpdate.Equal(deleteTime) {
			t.Errorf("cache must receive the soft deleted row. got: %+v", got)
		}
		return nil
	})
	if err := h.SummaryDelete(ctx, id); err != nil {
		t.Fatalf("delete failed. err: %v", err)
	}

	var bystanderMarked bool
	if errQuery := dbTest.QueryRow(`SELECT tm_delete IS NOT NULL OR tm_update IS NOT NULL FROM ai_summaries WHERE id = ?`, bystanderID.Bytes()).Scan(&bystanderMarked); errQuery != nil {
		t.Fatalf("could not read the bystander row. err: %v", errQuery)
	}
	if bystanderMarked {
		t.Errorf("deleting one summary must not touch another")
	}

	// observed current behavior: the row is kept and only marked, so a read
	// after a cache miss still returns it with tm_delete set.
	mockCache.EXPECT().SummaryGet(ctx, id).Return(nil, errors.New("miss"))
	mockCache.EXPECT().SummarySet(ctx, gomock.Any()).Return(nil)
	res, err := h.SummaryGet(ctx, id)
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if res.TMDelete == nil || !res.TMDelete.Equal(deleteTime) {
		t.Errorf("tm_delete mismatch. got: %v", res.TMDelete)
	}
}
