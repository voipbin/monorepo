package dispatchhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

func Test_tick(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &dispatchHandler{
		utilHandler:     mockUtil,
		db:              mockDB,
		sem:             make(chan struct{}, 1),
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	ctx := context.Background()

	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	uninit := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("e8f90a1b-1a13-11ef-8f0f-6f2a2b7f9d3e"),
		},
		Name:    "test-uninit",
		Cron:    "0 1 * * *",
		Enabled: true,
	}

	expectNext, err := schedule.NextRun(uninit.Cron, now)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	mockUtil.EXPECT().TimeNow().Return(&now)
	mockDB.EXPECT().ExecutionReapAbandoned(ctx, now).Return(int64(2), nil)
	mockDB.EXPECT().ExecutionCountAll(ctx).Return(int64(5), nil)
	mockDB.EXPECT().ScheduleListUninitialized(ctx, uint64(scanLimit)).Return([]*schedule.Schedule{uninit}, nil)
	mockDB.EXPECT().ScheduleInitNextRun(ctx, uninit.ID, expectNext).Return(true, nil)
	mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return([]*schedule.Schedule{}, nil)

	beforeAbandoned := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)))

	h.tick(ctx)

	afterAbandoned := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)))
	if afterAbandoned-beforeAbandoned != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", afterAbandoned-beforeAbandoned)
	}

	if rows := testutil.ToFloat64(promExecutionRows); rows != 5 {
		t.Errorf("Wrong match. expect: 5, got: %v", rows)
	}
}

func Test_Run_stops_on_context_cancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &dispatchHandler{
		utilHandler:     mockUtil,
		db:              mockDB,
		tickInterval:    10 * time.Millisecond,
		sem:             make(chan struct{}, 1),
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockDB.EXPECT().ExecutionReapAbandoned(gomock.Any(), now).Return(int64(0), nil).AnyTimes()
	mockDB.EXPECT().ExecutionCountAll(gomock.Any()).Return(int64(0), nil).AnyTimes()
	mockDB.EXPECT().ScheduleListUninitialized(gomock.Any(), uint64(scanLimit)).Return([]*schedule.Schedule{}, nil).AnyTimes()
	mockDB.EXPECT().ScheduleListDue(gomock.Any(), now, uint64(scanLimit)).Return([]*schedule.Schedule{}, nil).AnyTimes()

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	done := make(chan struct{})
	go func() {
		h.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("Wrong match. expect: Run returned after cancel, got: timeout")
	}
}
