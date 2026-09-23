package dbhandler

import (
	"context"
	"testing"
	"time"

	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
)

// Test_QueuecallListOldestWaiting verifies the FIFO oldest-waiting query
// (VOIP-1539 §3.5, event entry point B): only StatusWaiting rows for the
// given queue are returned, oldest tm_create first, and only up to limit.
func Test_QueuecallListOldestWaiting(t *testing.T) {
	queueID := uuid.FromStringOrNil("10b6bd90-b49d-11ec-950c-d3213b7e8cda")
	otherQueueID := uuid.FromStringOrNil("20b6bd90-b49d-11ec-950c-d3213b7e8cda")

	oldest := uuid.FromStringOrNil("a1111111-1111-1111-1111-111111111111")
	middle := uuid.FromStringOrNil("a2222222-2222-2222-2222-222222222222")
	newest := uuid.FromStringOrNil("a3333333-3333-3333-3333-333333333333")
	notWaiting := uuid.FromStringOrNil("a4444444-4444-4444-4444-444444444444")
	otherQueue := uuid.FromStringOrNil("a5555555-5555-5555-5555-555555555555")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}
	ctx := context.Background()

	create := func(id uuid.UUID, qID uuid.UUID, status queuecall.Status, tmCreate time.Time) {
		mockUtil.EXPECT().TimeNow().Return(&tmCreate)
		mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil)
		qc := &queuecall.Queuecall{
			Identity: commonidentity.Identity{ID: id},
			QueueID:  qID,
			Status:   status,
		}
		if err := h.QueuecallCreate(ctx, qc); err != nil {
			t.Fatalf("Could not create the queuecall for setup. err: %v", err)
		}
	}

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	create(newest, queueID, queuecall.StatusWaiting, base.Add(3*time.Second))
	create(oldest, queueID, queuecall.StatusWaiting, base.Add(1*time.Second))
	create(middle, queueID, queuecall.StatusWaiting, base.Add(2*time.Second))
	create(notWaiting, queueID, queuecall.StatusService, base.Add(500*time.Millisecond))
	create(otherQueue, otherQueueID, queuecall.StatusWaiting, base)

	res, err := h.QueuecallListOldestWaiting(ctx, queueID, 2)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2, got: %d", len(res))
	}
	if res[0].ID != oldest {
		t.Errorf("Wrong match. expect: %v, got: %v", oldest, res[0].ID)
	}
	if res[1].ID != middle {
		t.Errorf("Wrong match. expect: %v, got: %v", middle, res[1].ID)
	}
}
