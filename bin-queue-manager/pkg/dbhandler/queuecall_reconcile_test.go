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

// Test_QueuecallListConnectingStale verifies the connecting-stale query
// (VOIP-1539 §5.2, matching backstop recovery A): only StatusConnecting rows
// with tm_update older than the given cutoff are returned, oldest-stale
// first, and only up to limit.
func Test_QueuecallListConnectingStale(t *testing.T) {
	stale1 := uuid.FromStringOrNil("b1111111-1111-1111-1111-111111111111")
	stale2 := uuid.FromStringOrNil("b2222222-2222-2222-2222-222222222222")
	fresh := uuid.FromStringOrNil("b3333333-3333-3333-3333-333333333333")
	notConnecting := uuid.FromStringOrNil("b4444444-4444-4444-4444-444444444444")

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

	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff := base.Add(10 * time.Minute)

	create := func(id uuid.UUID, status queuecall.Status, tmUpdate time.Time) {
		tmCreate := tmUpdate
		mockUtil.EXPECT().TimeNow().Return(&tmCreate)
		mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		qc := &queuecall.Queuecall{
			Identity: commonidentity.Identity{ID: id},
			Status:   status,
		}
		if err := h.QueuecallCreate(ctx, qc); err != nil {
			t.Fatalf("Could not create the queuecall for setup. err: %v", err)
		}

		// QueuecallCreate leaves tm_update NULL (only later status-transition
		// writes stamp it) -- set it explicitly so the WHERE tm_update <
		// cutoff clause below has something to compare against.
		// QueuecallUpdate unconditionally overwrites FieldTMUpdate with
		// utilHandler.TimeNow() (queuecall.go:403), so the explicit
		// FieldTMUpdate value passed below is discarded -- stub TimeNow to
		// return the intended timestamp instead.
		mockUtil.EXPECT().TimeNow().Return(&tmUpdate)
		if err := h.QueuecallUpdate(ctx, id, map[queuecall.Field]any{
			queuecall.FieldTMUpdate: tmUpdate,
		}); err != nil {
			t.Fatalf("Could not set tm_update for setup. err: %v", err)
		}
	}

	// stale1/stale2 are stamped well before cutoff; fresh is stamped after.
	create(stale1, queuecall.StatusConnecting, base.Add(1*time.Minute))
	create(stale2, queuecall.StatusConnecting, base.Add(2*time.Minute))
	create(fresh, queuecall.StatusConnecting, base.Add(20*time.Minute))
	create(notConnecting, queuecall.StatusWaiting, base.Add(1*time.Minute))

	res, err := h.QueuecallListConnectingStale(ctx, cutoff, 10)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("Wrong match. expect: 2, got: %d", len(res))
	}
	if res[0].ID != stale1 {
		t.Errorf("Wrong match. expect: %v, got: %v", stale1, res[0].ID)
	}
	if res[1].ID != stale2 {
		t.Errorf("Wrong match. expect: %v, got: %v", stale2, res[1].ID)
	}
}

// Test_QueuecallListConnectingStale_limit verifies the limit is honored.
func Test_QueuecallListConnectingStale_limit(t *testing.T) {
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

	base := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	cutoff := base.Add(time.Hour)

	create := func(id uuid.UUID, tmUpdate time.Time) {
		tmCreate := tmUpdate
		mockUtil.EXPECT().TimeNow().Return(&tmCreate)
		mockCache.EXPECT().QueuecallSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		qc := &queuecall.Queuecall{
			Identity: commonidentity.Identity{ID: id},
			Status:   queuecall.StatusConnecting,
		}
		if err := h.QueuecallCreate(ctx, qc); err != nil {
			t.Fatalf("Could not create the queuecall for setup. err: %v", err)
		}
		mockUtil.EXPECT().TimeNow().Return(&tmUpdate)
		if err := h.QueuecallUpdate(ctx, id, map[queuecall.Field]any{
			queuecall.FieldTMUpdate: tmUpdate,
		}); err != nil {
			t.Fatalf("Could not set tm_update for setup. err: %v", err)
		}
	}

	create(uuid.FromStringOrNil("c1111111-1111-1111-1111-111111111111"), base.Add(1*time.Minute))
	create(uuid.FromStringOrNil("c2222222-2222-2222-2222-222222222222"), base.Add(2*time.Minute))
	create(uuid.FromStringOrNil("c3333333-3333-3333-3333-333333333333"), base.Add(3*time.Minute))

	res, err := h.QueuecallListConnectingStale(ctx, cutoff, 2)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 2 {
		t.Errorf("Wrong match. expect: 2, got: %d", len(res))
	}
}

// Test_QueuecallListWaitingOldest verifies the cross-queue waiting sweep
// (VOIP-1539 §5.2, matching backstop recovery B): only StatusWaiting rows
// are returned, oldest tm_create first, across all queues, up to limit. Uses
// QueuecallList (the same DB handle other tests share via the in-memory
// sqlite `cache=shared` DSN) with a customer_id filter to stay isolated from
// rows other tests in this package may have already inserted -- a raw
// cross-queue scan has no such natural isolation key of its own, so the
// production query's own filter (status=waiting only) is verified here by
// asserting on rows created within this test rather than an exact result
// length across the whole shared table.
func Test_QueuecallListWaitingOldest(t *testing.T) {
	customerID := uuid.FromStringOrNil("d9000000-0000-0000-0000-000000000099")
	queueA := uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000001")
	queueB := uuid.FromStringOrNil("d0000000-0000-0000-0000-000000000002")

	oldest := uuid.FromStringOrNil("e1111111-1111-1111-1111-111111111111")
	middle := uuid.FromStringOrNil("e2222222-2222-2222-2222-222222222222")
	newest := uuid.FromStringOrNil("e3333333-3333-3333-3333-333333333333")
	notWaiting := uuid.FromStringOrNil("e4444444-4444-4444-4444-444444444444")

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
			Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
			QueueID:  qID,
			Status:   status,
		}
		if err := h.QueuecallCreate(ctx, qc); err != nil {
			t.Fatalf("Could not create the queuecall for setup. err: %v", err)
		}
	}

	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	create(newest, queueA, queuecall.StatusWaiting, base.Add(3*time.Second))
	create(oldest, queueB, queuecall.StatusWaiting, base.Add(1*time.Second))
	create(middle, queueA, queuecall.StatusWaiting, base.Add(2*time.Second))
	create(notWaiting, queueA, queuecall.StatusService, base.Add(500*time.Millisecond))

	// The production query itself is a global (unfiltered by customer)
	// oldest-waiting scan, so pull a generous page and assert this test's
	// own rows appear in the right relative (oldest-first) order rather
	// than asserting an exact page length against a table other tests in
	// this package also write to.
	res, err := h.QueuecallListWaitingOldest(ctx, 1000)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	var gotOldest, gotMiddle, gotNewest *queuecall.Queuecall
	for _, r := range res {
		switch r.ID {
		case oldest:
			gotOldest = r
		case middle:
			gotMiddle = r
		case newest:
			gotNewest = r
		case notWaiting:
			t.Errorf("StatusService queuecall must not appear in QueuecallListWaitingOldest. id: %v", notWaiting)
		}
	}

	if gotOldest == nil || gotMiddle == nil || gotNewest == nil {
		t.Fatalf("Wrong match. expected all three waiting queuecalls present. oldest: %v, middle: %v, newest: %v", gotOldest, gotMiddle, gotNewest)
	}

	// oldest-first ordering: find each row's index and assert oldest <
	// middle < newest.
	indexOf := func(id uuid.UUID) int {
		for i, r := range res {
			if r.ID == id {
				return i
			}
		}
		return -1
	}
	if indexOf(oldest) >= indexOf(middle) || indexOf(middle) >= indexOf(newest) {
		t.Errorf("Wrong order. expect oldest < middle < newest, got indices oldest=%d middle=%d newest=%d", indexOf(oldest), indexOf(middle), indexOf(newest))
	}
}
