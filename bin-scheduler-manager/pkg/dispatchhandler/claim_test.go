package dispatchhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

func Test_claimAndDispatch_lock_busy(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &dispatchHandler{
		cache:           mockCache,
		db:              mockDB,
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	ctx := context.Background()

	slot := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("0a7f7c6a-1a10-11ef-8f0f-6f2a2b7f9d3e"),
		},
		Name:      "test-lock-busy",
		Cron:      "0 1 * * *",
		TMNextRun: &slot,
	}

	mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(nil, cachehandler.ErrLockBusy)

	before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedLock))

	h.claimAndDispatch(ctx, s)

	after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedLock))
	if after-before != 1 {
		t.Errorf("Wrong match. expect: 1, got: %v", after-before)
	}
}

func Test_claimAndDispatch_overlap_once_per_slot(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &dispatchHandler{
		cache:           mockCache,
		db:              mockDB,
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	ctx := context.Background()

	slot := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("1b2c3d4e-1a10-11ef-9d3e-6f2a2b7f9d3e"),
		},
		Name:      "test-overlap-once",
		Cron:      "0 1 * * *",
		TMNextRun: &slot,
	}

	unlock := func() {}
	mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(unlock, nil).Times(3)
	mockDB.EXPECT().ExecutionHasRunning(ctx, s.ID).Return(true, nil).Times(3)

	before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap))

	// same slot skipped twice — counted once
	h.claimAndDispatch(ctx, s)
	h.claimAndDispatch(ctx, s)

	after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap))
	if after-before != 1 {
		t.Errorf("Wrong match. expect: 1, got: %v", after-before)
	}

	// a new slot is counted again
	nextSlot := time.Date(2026, 8, 2, 1, 0, 0, 0, time.UTC)
	s.TMNextRun = &nextSlot
	h.claimAndDispatch(ctx, s)

	after = testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap))
	if after-before != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", after-before)
	}
}

func Test_claimAndDispatch_cas_lost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &dispatchHandler{
		utilHandler:     mockUtil,
		cache:           mockCache,
		db:              mockDB,
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	ctx := context.Background()

	now := time.Date(2026, 8, 1, 1, 0, 5, 0, time.UTC)
	slot := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2c3d4e5f-1a10-11ef-a06f-6f2a2b7f9d3e"),
		},
		Name:      "test-cas-lost",
		Cron:      "0 1 * * *",
		TMNextRun: &slot,
	}

	expectNext, err := schedule.NextRun(s.Cron, now)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	unlock := func() {}
	mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(unlock, nil)
	mockDB.EXPECT().ExecutionHasRunning(ctx, s.ID).Return(false, nil)
	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, s, expectNext, now, execution.TriggerTypeCron, slot).Return(nil, nil)

	before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedCAS))

	h.claimAndDispatch(ctx, s)

	after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedCAS))
	if after-before != 1 {
		t.Errorf("Wrong match. expect: 1, got: %v", after-before)
	}
}

func Test_claimAndDispatch_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	slot := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 1, 1, 0, 5, 0, time.UTC)

	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("3d4e5f60-1a10-11ef-b1cd-6f2a2b7f9d3e"),
			CustomerID: uuid.FromStringOrNil("6c73ff34-7f4c-11ec-b4d5-5b94d40e4071"),
		},
		Name: "test-claim-success",
		Cron: "0 1 * * *",

		TargetQueue:    "bin-manager.number-manager.request",
		TargetURI:      "/v1/numbers/renew",
		TargetMethod:   "POST",
		TargetDataType: "application/json",
		TargetData:     []byte(`{"days":28}`),

		TimeoutMS: 300000,
		RetryMax:  0,
		Enabled:   true,

		TMNextRun: &slot,
	}

	exec := &execution.Execution{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("4e5f6071-1a10-11ef-8ff5-6f2a2b7f9d3e"),
		},
		ScheduleID:  s.ID,
		TriggerType: execution.TriggerTypeCron,
		Status:      execution.StatusRunning,
	}

	h := &dispatchHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		db:            mockDB,
		cache:         mockCache,
		notifyHandler: mockNotify,
		retryBackoff:  time.Millisecond,
		lastOverlapSkip: map[uuid.UUID]time.Time{
			s.ID: slot, // pre-set to prove a successful claim clears the marker
		},
	}

	ctx := context.Background()

	expectNext, err := schedule.NextRun(s.Cron, now)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	unlock := func() {}
	mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(unlock, nil)
	mockDB.EXPECT().ExecutionHasRunning(ctx, s.ID).Return(false, nil)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, s, expectNext, now, execution.TriggerTypeCron, slot).Return(exec, nil)

	mockReq.EXPECT().SendRequest(ctx, gomock.Any(), s.TargetURI, sock.RequestMethodPost, s.TimeoutMS, 0, s.TargetDataType, gomock.Any()).Return(&sock.Response{StatusCode: 200, Data: []byte(`{"ok":true}`)}, nil)
	mockDB.EXPECT().ExecutionComplete(ctx, exec.ID, execution.StatusSuccess, 200, "", `{"ok":true}`, 1, 0, now).Return(true, nil)
	mockDB.EXPECT().ExecutionGet(ctx, exec.ID).Return(exec, nil)
	mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeExecutionSucceeded, exec)

	h.claimAndDispatch(ctx, s)

	if _, ok := h.lastOverlapSkip[s.ID]; ok {
		t.Errorf("Wrong match. expect: overlap-skip marker cleared, got: still set")
	}
}
