package queuecallhandler

import (
	"context"
	"errors"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/cachehandler"
	"monorepo/bin-queue-manager/pkg/dbhandler"
	"monorepo/bin-queue-manager/pkg/queuehandler"
)

// Test_Reconcile_lockBusy verifies the matching backstop's per-tick lease
// (VOIP-1539 §5.2 step 1): when another replica already holds the lease,
// this pass does nothing else -- no DB scan, no release attempt.
func Test_Reconcile_lockBusy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler: mockUtil,
		cache:       mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(false, nil)
	// No further mock expectations: a busy lease must skip the whole pass
	// (no DB scan, no lease release).

	h.Reconcile(ctx)
}

// Test_Reconcile_lockAcquireError verifies a Redis error acquiring the lease
// is treated the same as "skip this tick" -- no DB scan, no release
// (there is nothing to release; the lease was never acquired).
func Test_Reconcile_lockAcquireError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler: mockUtil,
		cache:       mockCache,
	}
	ctx := context.Background()

	mockCache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(false, errors.New("redis down"))

	h.Reconcile(ctx)
}

// Test_Reconcile_acquiredAndReleased verifies the happy path (VOIP-1539
// §5.2): a successful lease acquire runs both recovery scans, and the
// lease is released afterward regardless of what the scans found.
func Test_Reconcile_acquiredAndReleased(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		cache:         mockCache,
		notifyhandler: mockNotify,
		queueHandler:  mockQueue,
	}
	ctx := context.Background()

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	mockCache.EXPECT().ReconcileLockAcquire(ctx, gomock.Any(), reconcileLockTTL).Return(true, nil)
	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{}, nil)
	mockDB.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{}, nil)
	mockCache.EXPECT().ReconcileLockRelease(ctx).Return(nil)

	h.Reconcile(ctx)
}

// Test_reconcileConnectingStale_rollsBackAndHangsUp verifies recovery A
// (VOIP-1539 §5.2 step 2): a stale connecting queuecall that wins the
// rollback CAS gets its groupcall hung up AFTER the CAS win (Round 3
// review fix) -- proving hangup is gated on this queuecall genuinely still
// being connecting at CAS time, not fired unconditionally off the
// pre-rollback snapshot.
func Test_reconcileConnectingStale_rollsBackAndHangsUp(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyhandler: mockNotify,
		queueHandler:  mockQueue,
	}
	ctx := context.Background()

	now := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	qcID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000001")
	groupcallID := uuid.FromStringOrNil("bb000000-0000-0000-0000-000000000001")
	qc := &queuecall.Queuecall{
		Identity:    commonidentity.Identity{ID: qcID},
		Status:      queuecall.StatusConnecting,
		GroupcallID: groupcallID,
	}

	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc}, nil)

	// CAS wins (affected=1) -- only then does hangup fire.
	mockDB.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qcID).Return(int64(1), nil)
	mockReq.EXPECT().CallV1GroupcallHangup(ctx, groupcallID).Return(nil, nil)
	mockDB.EXPECT().QueuecallGet(ctx, qcID).Return(qc, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, qc.CustomerID, queuecall.EventTypeQueuecallWaiting, qc)
	mockQueue.EXPECT().AddWaitQueueCallID(gomock.Any(), qc.QueueID, qc.ID).Return(&queue.Queue{}, nil).AnyTimes()

	h.reconcileConnectingStale(ctx)
}

// Test_reconcileConnectingStale_casLostSkipsHangup verifies the exact
// defect Round 3 review flagged: when the rollback CAS loses (the
// queuecall already progressed past connecting on its own, e.g. a join
// landed between the stale-list snapshot and this call), the groupcall
// must NOT be hung up -- it may be a real, already-connected call.
func Test_reconcileConnectingStale_casLostSkipsHangup(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyhandler: mockNotify,
	}
	ctx := context.Background()

	now := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	qcID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000003")
	groupcallID := uuid.FromStringOrNil("bb000000-0000-0000-0000-000000000003")
	// This snapshot is stale by the time the CAS runs: qc still shows
	// GroupcallID set (as it was when listed), but the queuecall has
	// actually already moved to service via a winning join in the
	// meantime.
	qc := &queuecall.Queuecall{
		Identity:    commonidentity.Identity{ID: qcID},
		Status:      queuecall.StatusConnecting,
		GroupcallID: groupcallID,
	}
	current := &queuecall.Queuecall{
		Identity:    commonidentity.Identity{ID: qcID},
		Status:      queuecall.StatusService,
		GroupcallID: groupcallID,
	}

	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc}, nil)

	// CAS loses (affected=0): no hangup, no reservation release, no
	// webhook, no requeue -- a pure no-op that just re-fetches current state.
	mockDB.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qcID).Return(int64(0), nil)
	mockDB.EXPECT().QueuecallGet(ctx, qcID).Return(current, nil)
	// No CallV1GroupcallHangup expectation: must not be called when the
	// CAS loses, even though qc.GroupcallID (the stale snapshot) is set.

	h.reconcileConnectingStale(ctx)
}

// Test_reconcileConnectingStale_noGroupcall verifies a stale connecting
// queuecall with no groupcall (GroupcallID == uuid.Nil) that wins the
// rollback CAS skips the hangup call entirely and proceeds straight to the
// winner-only side effects.
func Test_reconcileConnectingStale_noGroupcall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	h := &queuecallHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		notifyhandler: mockNotify,
		queueHandler:  mockQueue,
	}
	ctx := context.Background()

	now := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	qcID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000002")
	qc := &queuecall.Queuecall{
		Identity: commonidentity.Identity{ID: qcID},
		Status:   queuecall.StatusConnecting,
		// GroupcallID left as uuid.Nil (zero value).
	}

	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc}, nil)
	// CAS wins (affected=1), but GroupcallID is Nil.
	mockDB.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qcID).Return(int64(1), nil)
	// No CallV1GroupcallHangup expectation: must not be called for a
	// zero-value GroupcallID even on a CAS win.
	mockDB.EXPECT().QueuecallGet(ctx, qcID).Return(qc, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, qc.CustomerID, queuecall.EventTypeQueuecallWaiting, qc)
	mockQueue.EXPECT().AddWaitQueueCallID(gomock.Any(), qc.QueueID, qc.ID).Return(&queue.Queue{}, nil).AnyTimes()

	h.reconcileConnectingStale(ctx)
}

// Test_reconcileWaiting_triesMatchForEach verifies recovery B (VOIP-1539
// §5.2 step 3): every waiting queuecall gets one matchWaitingQueuecall
// attempt, and a queue with no available agents is a no-op (proving the
// existing entry-point-A logic is reused as-is, not duplicated).
func Test_reconcileWaiting_triesMatchForEach(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockQueue := queuehandler.NewMockQueueHandler(mc)

	h := &queuecallHandler{
		db:           mockDB,
		queueHandler: mockQueue,
	}
	ctx := context.Background()

	qc1ID := uuid.FromStringOrNil("cc000000-0000-0000-0000-000000000001")
	qc2ID := uuid.FromStringOrNil("cc000000-0000-0000-0000-000000000002")
	queueID := uuid.FromStringOrNil("dd000000-0000-0000-0000-000000000001")
	qc1 := &queuecall.Queuecall{Identity: commonidentity.Identity{ID: qc1ID}, QueueID: queueID}
	qc2 := &queuecall.Queuecall{Identity: commonidentity.Identity{ID: qc2ID}, QueueID: queueID}

	mockDB.EXPECT().QueuecallListWaitingOldest(ctx, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc1, qc2}, nil)
	// matchWaitingQueuecall's first step is GetAgents; returning an error
	// for both proves each waiting queuecall gets its own attempt without
	// asserting on Execute's full RPC chain (already covered by the
	// Test_Execute* suite in execute_test.go).
	mockQueue.EXPECT().GetAgents(ctx, queueID, gomock.Any()).Return(nil, errors.New("agent-manager down")).Times(2)

	h.reconcileWaiting(ctx)
}
