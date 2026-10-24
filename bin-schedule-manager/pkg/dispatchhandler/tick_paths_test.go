package dispatchhandler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

const testWait = 5 * time.Second

// waitSlotsReleased proves that every spawned dispatch goroutine has finished:
// the test acquires all semaphore slots, which only succeeds once each
// goroutine has run its deferred release.
func waitSlotsReleased(t *testing.T, h *dispatchHandler) {
	t.Helper()

	timeout := time.After(testWait)
	for i := 0; i < cap(h.sem); i++ {
		select {
		case h.sem <- struct{}{}:
		case <-timeout:
			t.Fatalf("Wrong match. expect: all semaphore slots released, got: timeout")
		}
	}
}

func newTickTestHandler(mc *gomock.Controller, concurrency int) (*dispatchHandler, *dbhandler.MockDBHandler, *cachehandler.MockCacheHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := &dispatchHandler{
		utilHandler:     utilhandler.NewMockUtilHandler(mc),
		db:              mockDB,
		cache:           mockCache,
		sem:             make(chan struct{}, concurrency),
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	return h, mockDB, mockCache
}

func newTickTestSchedule(id string, name string, next *time.Time) *schedule.Schedule {
	return &schedule.Schedule{
		Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil(id)},
		Name:      name,
		Cron:      "0 1 * * *",
		Enabled:   true,
		TMNextRun: next,
	}
}

func Test_dispatchDue_lag_gauge(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)
	late := now.Add(-30 * time.Second)
	exact := now
	future := now.Add(time.Hour)

	tests := []struct {
		name   string
		next   *time.Time
		preset float64

		expectLock bool
		expectLag  float64
	}{
		{name: "late by 30 seconds", next: &late, preset: 5, expectLock: true, expectLag: 30},
		{name: "exactly due", next: &exact, preset: 5, expectLock: true, expectLag: 0},
		{name: "next run in the future is floored at zero", next: &future, preset: 100, expectLock: true, expectLag: 0},
		{name: "nil next run leaves the gauge untouched", next: nil, preset: 7, expectLock: false, expectLag: 7},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockCache := newTickTestHandler(mc, 2)
			ctx := context.Background()

			s := newTickTestSchedule("a1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d3e", fmt.Sprintf("lag-gauge-case-%d", i), tt.next)
			promScheduleLagSeconds.WithLabelValues(s.Name).Set(tt.preset)

			mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return([]*schedule.Schedule{s}, nil)
			if tt.expectLock {
				mockCache.EXPECT().LockSchedule(gomock.Any(), s.ID, scheduleLockTTL).Return(nil, cachehandler.ErrLockBusy)
			}

			h.dispatchDue(ctx, now)
			waitSlotsReleased(t, h)

			if got := testutil.ToFloat64(promScheduleLagSeconds.WithLabelValues(s.Name)); got != tt.expectLag {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectLag, got)
			}
		})
	}
}

func Test_dispatchDue_list_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTickTestHandler(mc, 1)
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	// no lock or execution call is expected after the list failure
	mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return(nil, fmt.Errorf("db down"))

	h.dispatchDue(ctx, now)
	waitSlotsReleased(t, h)
}

func Test_dispatchDue_canceled_context_with_full_semaphore(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTickTestHandler(mc, 1)
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)
	next := now.Add(-30 * time.Second)

	first := newTickTestSchedule("a1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d01", "cancel-case-first", &next)
	second := newTickTestSchedule("a1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d02", "cancel-case-second", &next)
	promScheduleLagSeconds.WithLabelValues(first.Name).Set(1)
	promScheduleLagSeconds.WithLabelValues(second.Name).Set(2)

	// the only slot is taken, so the select can only leave through ctx.Done
	h.sem <- struct{}{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// a dispatch would call the cache, which has no expectation and fails the test
	mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return([]*schedule.Schedule{first, second}, nil)

	h.dispatchDue(ctx, now)

	if got := testutil.ToFloat64(promScheduleLagSeconds.WithLabelValues(first.Name)); got != 30 {
		t.Errorf("Wrong match. expect: 30, got: %v", got)
	}
	// the loop stopped at the first schedule, so the second keeps its value
	if got := testutil.ToFloat64(promScheduleLagSeconds.WithLabelValues(second.Name)); got != 2 {
		t.Errorf("Wrong match. expect: 2, got: %v", got)
	}
}

func Test_dispatchDue_concurrency_cap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	const capacity = 2
	h, mockDB, mockCache := newTickTestHandler(mc, capacity)
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)
	next := now.Add(-time.Second)

	ids := []string{
		"b1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d01",
		"b1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d02",
		"b1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d03",
		"b1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d04",
		"b1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d05",
	}
	schedules := []*schedule.Schedule{}
	for i, id := range ids {
		schedules = append(schedules, newTickTestSchedule(id, fmt.Sprintf("cap-case-%d", i), &next))
	}

	var inFlight, maxInFlight int32
	entered := make(chan struct{}, len(ids))
	gate := make(chan struct{})

	mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return(schedules, nil)
	mockCache.EXPECT().LockSchedule(gomock.Any(), gomock.Any(), scheduleLockTTL).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ time.Duration) (func(), error) {
			cur := atomic.AddInt32(&inFlight, 1)
			for {
				old := atomic.LoadInt32(&maxInFlight)
				if cur <= old || atomic.CompareAndSwapInt32(&maxInFlight, old, cur) {
					break
				}
			}
			entered <- struct{}{}
			<-gate
			atomic.AddInt32(&inFlight, -1)

			return nil, cachehandler.ErrLockBusy
		}).Times(len(ids))

	done := make(chan struct{})
	go func() {
		h.dispatchDue(ctx, now)
		close(done)
	}()

	for i := 0; i < capacity; i++ {
		select {
		case <-entered:
		case <-time.After(testWait):
			t.Fatalf("Wrong match. expect: %d dispatches started, got: timeout", capacity)
		}
	}

	// the semaphore is full, so a further dispatch must not start (bounded
	// absence check: only a failure depends on the timing)
	select {
	case <-entered:
		t.Errorf("Wrong match. expect: no dispatch beyond the cap, got: an extra dispatch")
	case <-time.After(100 * time.Millisecond):
	}

	close(gate)

	select {
	case <-done:
	case <-time.After(testWait):
		t.Fatalf("Wrong match. expect: dispatchDue returned, got: timeout")
	}
	waitSlotsReleased(t, h)

	if got := atomic.LoadInt32(&maxInFlight); got != capacity {
		t.Errorf("Wrong match. expect max in flight: %d, got: %d", capacity, got)
	}
}

func Test_initUninitialized(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	valid := newTickTestSchedule("c1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d01", "init-valid", nil)
	invalid := newTickTestSchedule("c1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d02", "init-invalid", nil)
	invalid.Cron = "not a cron spec"
	failing := newTickTestSchedule("c1b2c3d4-1a13-11ef-8f0f-6f2a2b7f9d03", "init-failing", nil)

	expectNext, err := schedule.NextRun(valid.Cron, now)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	t.Run("list error makes no init call", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		mockDB.EXPECT().ScheduleListUninitialized(ctx, uint64(scanLimit)).Return(nil, fmt.Errorf("db down"))

		h.initUninitialized(ctx, now)
	})

	t.Run("empty list makes no init call", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		mockDB.EXPECT().ScheduleListUninitialized(ctx, uint64(scanLimit)).Return([]*schedule.Schedule{}, nil)

		h.initUninitialized(ctx, now)
	})

	t.Run("invalid cron and init error do not stop the later schedules", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		expectFailingNext, errFailing := schedule.NextRun(failing.Cron, now)
		if errFailing != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", errFailing)
		}

		mockDB.EXPECT().ScheduleListUninitialized(ctx, uint64(scanLimit)).Return([]*schedule.Schedule{invalid, failing, valid}, nil)
		// the invalid cron makes no init call
		mockDB.EXPECT().ScheduleInitNextRun(ctx, failing.ID, expectFailingNext).Return(false, fmt.Errorf("db down"))
		mockDB.EXPECT().ScheduleInitNextRun(ctx, valid.ID, expectNext).Return(true, nil)

		h.initUninitialized(ctx, now)
	})
}

func Test_reapAbandoned_and_refreshExecutionRows_paths(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	t.Run("reap error leaves the abandoned counter", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		before := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)))
		mockDB.EXPECT().ExecutionReapAbandoned(ctx, now).Return(int64(0), fmt.Errorf("db down"))

		h.reapAbandoned(ctx, now)

		if got := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned))); got != before {
			t.Errorf("Wrong match. expect: %v, got: %v", before, got)
		}
	})

	t.Run("reap count zero leaves the abandoned counter", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		before := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)))
		mockDB.EXPECT().ExecutionReapAbandoned(ctx, now).Return(int64(0), nil)

		h.reapAbandoned(ctx, now)

		if got := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned))); got != before {
			t.Errorf("Wrong match. expect: %v, got: %v", before, got)
		}
	})

	t.Run("reap count one increments the abandoned counter by one", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		before := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)))
		mockDB.EXPECT().ExecutionReapAbandoned(ctx, now).Return(int64(1), nil)

		h.reapAbandoned(ctx, now)

		if got := testutil.ToFloat64(promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned))); got != before+1 {
			t.Errorf("Wrong match. expect: %v, got: %v", before+1, got)
		}
	})

	t.Run("count error leaves the gauge", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockDB, _ := newTickTestHandler(mc, 1)
		ctx := context.Background()

		promExecutionRows.Set(7)
		mockDB.EXPECT().ExecutionCountAll(ctx).Return(int64(0), fmt.Errorf("db down"))

		h.refreshExecutionRows(ctx)

		if got := testutil.ToFloat64(promExecutionRows); got != 7 {
			t.Errorf("Wrong match. expect: 7, got: %v", got)
		}
	})
}

func Test_tick_error_isolation(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newTickTestHandler(mc, 1)
	mockUtil := h.utilHandler.(*utilhandler.MockUtilHandler)
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 0, 30, 0, 0, time.UTC)

	// every step fails, each later step still runs, in the documented order
	mockUtil.EXPECT().TimeNow().Return(&now)
	gomock.InOrder(
		mockDB.EXPECT().ExecutionReapAbandoned(ctx, now).Return(int64(0), fmt.Errorf("db down")),
		mockDB.EXPECT().ExecutionCountAll(ctx).Return(int64(0), fmt.Errorf("db down")),
		mockDB.EXPECT().ScheduleListUninitialized(ctx, uint64(scanLimit)).Return(nil, fmt.Errorf("db down")),
		mockDB.EXPECT().ScheduleListDue(ctx, now, uint64(scanLimit)).Return(nil, fmt.Errorf("db down")),
	)

	h.tick(ctx)
}
