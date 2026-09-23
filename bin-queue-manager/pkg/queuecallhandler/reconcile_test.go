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
// (VOIP-1539 §5.2 step 2): a stale connecting queuecall with a live
// groupcall gets that groupcall hung up, then rolled back to waiting.
func Test_reconcileConnectingStale_rollsBackAndHangsUp(t *testing.T) {
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
	qcID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000001")
	groupcallID := uuid.FromStringOrNil("bb000000-0000-0000-0000-000000000001")
	qc := &queuecall.Queuecall{
		Identity:    commonidentity.Identity{ID: qcID},
		Status:      queuecall.StatusConnecting,
		GroupcallID: groupcallID,
	}

	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc}, nil)
	mockReq.EXPECT().CallV1GroupcallHangup(ctx, groupcallID).Return(nil, nil)

	// UpdateStatusWaitingRollback's own internals (CAS setter, cache
	// refresh, requeue) are exercised by pkg/queuecallhandler/db_test.go --
	// here it's invoked through the exported method, so the CAS setter call
	// alone (affected=0, no side effects) is enough to prove the wiring.
	mockDB.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qcID).Return(int64(0), nil)
	mockDB.EXPECT().QueuecallGet(ctx, qcID).Return(qc, nil)

	h.reconcileConnectingStale(ctx)
}

// Test_reconcileConnectingStale_noGroupcall verifies a stale connecting
// queuecall with no groupcall (GroupcallID == uuid.Nil) skips the hangup
// call and goes straight to rollback.
func Test_reconcileConnectingStale_noGroupcall(t *testing.T) {
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
	qcID := uuid.FromStringOrNil("aa000000-0000-0000-0000-000000000002")
	qc := &queuecall.Queuecall{
		Identity: commonidentity.Identity{ID: qcID},
		Status:   queuecall.StatusConnecting,
		// GroupcallID left as uuid.Nil (zero value).
	}

	mockUtil.EXPECT().TimeNowAdd(-connectingStaleAfter).Return(&now)
	mockDB.EXPECT().QueuecallListConnectingStale(ctx, now, uint64(reconcileScanLimit)).Return([]*queuecall.Queuecall{qc}, nil)
	// No CallV1GroupcallHangup expectation: must not be called for a
	// zero-value GroupcallID.
	mockDB.EXPECT().QueuecallSetStatusWaitingIfConnecting(ctx, qcID).Return(int64(0), nil)
	mockDB.EXPECT().QueuecallGet(ctx, qcID).Return(qc, nil)

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
