package dispatchhandler

import (
	"context"
	stderrors "errors"
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

	"monorepo/bin-schedule-manager/models/execution"
	"monorepo/bin-schedule-manager/models/schedule"
	"monorepo/bin-schedule-manager/pkg/cachehandler"
	"monorepo/bin-schedule-manager/pkg/dbhandler"
)

func Test_claimAndDispatch_error_paths(t *testing.T) {
	slot := time.Date(2026, 8, 1, 1, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 1, 1, 0, 5, 0, time.UTC)
	errBoom := stderrors.New("synthetic failure")

	tests := []struct {
		name string

		cron string

		lockErr      error
		runningErr   error
		claimErr     error
		expectUnlock int
	}{
		{
			name: "lock error other than busy",

			cron:    "0 1 * * *",
			lockErr: errBoom,

			expectUnlock: 0,
		},
		{
			name: "running check error",

			cron:       "0 1 * * *",
			runningErr: errBoom,

			expectUnlock: 1,
		},
		{
			name: "invalid cron",

			cron: "not a cron",

			expectUnlock: 1,
		},
		{
			name: "claim error",

			cron:     "0 1 * * *",
			claimErr: errBoom,

			expectUnlock: 1,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			// reqHandler and notifyHandler stay nil: any dispatch would panic
			h := &dispatchHandler{
				utilHandler:     mockUtil,
				cache:           mockCache,
				db:              mockDB,
				lastOverlapSkip: map[uuid.UUID]time.Time{},
			}

			ctx := context.Background()
			s := &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.Must(uuid.NewV4()),
				},
				Name:      "test-claim-error-" + string(rune('a'+i)),
				Cron:      tt.cron,
				TMNextRun: &slot,
			}

			unlockCount := 0
			unlock := func() { unlockCount++ }

			if tt.lockErr != nil {
				mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(nil, tt.lockErr)
			} else {
				mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(unlock, nil)
				mockDB.EXPECT().ExecutionHasRunning(ctx, s.ID).Return(false, tt.runningErr)
			}

			if tt.lockErr == nil && tt.runningErr == nil {
				mockUtil.EXPECT().TimeNow().Return(&now)
			}
			if tt.claimErr != nil {
				mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, s, gomock.Any(), now, execution.TriggerTypeCron, slot).Return(nil, tt.claimErr)
			}

			h.claimAndDispatch(ctx, s)

			if unlockCount != tt.expectUnlock {
				t.Errorf("Wrong match. expect unlock calls: %d, got: %d", tt.expectUnlock, unlockCount)
			}
			if len(h.lastOverlapSkip) != 0 {
				t.Errorf("Wrong match. expect: no overlap marker, got: %v", h.lastOverlapSkip)
			}
		})
	}
}

func Test_countOverlapSkip_nil_next_run(t *testing.T) {
	h := &dispatchHandler{lastOverlapSkip: map[uuid.UUID]time.Time{}}

	s := &schedule.Schedule{
		Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		Name:     "test-overlap-nil-next-run",
	}

	before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap))

	h.countOverlapSkip(s)

	after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap))
	if after != before {
		t.Errorf("Wrong match. expect: unchanged counter, got delta: %v", after-before)
	}
	if len(h.lastOverlapSkip) != 0 {
		t.Errorf("Wrong match. expect: no marker, got: %v", h.lastOverlapSkip)
	}
}

type dispatchFixture struct {
	h          *dispatchHandler
	mockReq    *requesthandler.MockRequestHandler
	mockDB     *dbhandler.MockDBHandler
	mockNotify *notifyhandler.MockNotifyHandler

	s    *schedule.Schedule
	exec *execution.Execution
	now  time.Time
}

func newDispatchFixture(mc *gomock.Controller, name string, retryMax int, backoff time.Duration) *dispatchFixture {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	now := time.Date(2026, 8, 1, 1, 0, 5, 0, time.UTC)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()

	s := &schedule.Schedule{
		Identity:       commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		Name:           name,
		TargetQueue:    "bin-manager.number-manager.request",
		TargetURI:      "/v1/numbers/renew",
		TargetMethod:   "POST",
		TargetDataType: "application/json",
		TargetData:     []byte(`{"days":28}`),
		TimeoutMS:      30000,
		RetryMax:       retryMax,
	}
	exec := &execution.Execution{
		Identity:   commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		ScheduleID: s.ID,
		Status:     execution.StatusRunning,
	}

	return &dispatchFixture{
		h: &dispatchHandler{
			utilHandler:     mockUtil,
			reqHandler:      mockReq,
			db:              mockDB,
			notifyHandler:   mockNotify,
			retryBackoff:    backoff,
			lastOverlapSkip: map[uuid.UUID]time.Time{},
		},
		mockReq:    mockReq,
		mockDB:     mockDB,
		mockNotify: mockNotify,
		s:          s,
		exec:       exec,
		now:        now,
	}
}

func Test_dispatch_negative_retry_max_sends_once(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	f := newDispatchFixture(mc, "test-dispatch-retry-minus-one", -1, time.Millisecond)
	ctx := context.Background()

	f.mockReq.EXPECT().SendRequest(ctx, gomock.Any(), f.s.TargetURI, gomock.Any(), f.s.TimeoutMS, 0, f.s.TargetDataType, gomock.Any()).
		Return(&sock.Response{StatusCode: 500}, nil).Times(1)
	f.mockDB.EXPECT().ExecutionComplete(ctx, f.exec.ID, execution.StatusFailed, 500, "dispatch failed with status code 500", "", 1, 0, f.now).Return(true, nil)
	f.mockDB.EXPECT().ExecutionGet(ctx, f.exec.ID).Return(f.exec, nil)
	f.mockNotify.EXPECT().PublishEvent(ctx, schedule.EventTypeExecutionFailed, f.exec)

	f.h.dispatch(ctx, f.s, f.exec)
}

func Test_dispatch_canceled_context_stops_retries(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the backoff is far longer than the test, so only the context can end the wait
	f := newDispatchFixture(mc, "test-dispatch-ctx-cancel", 3, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f.mockReq.EXPECT().SendRequest(ctx, gomock.Any(), f.s.TargetURI, gomock.Any(), f.s.TimeoutMS, 0, f.s.TargetDataType, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ any, _ string, _ any, _ int, _ int, _ string, _ []byte) (*sock.Response, error) {
			cancel()
			return nil, stderrors.New("synthetic send failure")
		}).Times(1)
	f.mockDB.EXPECT().ExecutionComplete(gomock.Any(), f.exec.ID, execution.StatusFailed, 0, "synthetic send failure", "", 1, 0, f.now).Return(true, nil)
	f.mockDB.EXPECT().ExecutionGet(gomock.Any(), f.exec.ID).Return(f.exec, nil)
	f.mockNotify.EXPECT().PublishEvent(gomock.Any(), schedule.EventTypeExecutionFailed, f.exec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		f.h.dispatch(ctx, f.s, f.exec)
	}()

	select {
	case <-done:
	case <-time.After(testWait):
		t.Fatalf("Wrong match. expect: dispatch returns after the context is canceled, got: timeout")
	}
}

func Test_dispatch_completion_errors(t *testing.T) {
	tests := []struct {
		name string

		completeErr error
		getErr      error

		expectGet   bool
		expectDelta float64
	}{
		{
			name: "complete error",

			completeErr: stderrors.New("synthetic complete failure"),

			expectGet:   false,
			expectDelta: 0,
		},
		{
			name: "get error after complete",

			getErr: stderrors.New("synthetic get failure"),

			expectGet:   true,
			expectDelta: 1,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			f := newDispatchFixture(mc, "test-dispatch-completion-error-"+string(rune('a'+i)), 0, time.Millisecond)
			ctx := context.Background()

			f.mockReq.EXPECT().SendRequest(ctx, gomock.Any(), f.s.TargetURI, gomock.Any(), f.s.TimeoutMS, 0, f.s.TargetDataType, gomock.Any()).
				Return(&sock.Response{StatusCode: 200, Data: []byte(`{}`)}, nil)

			if tt.completeErr != nil {
				f.mockDB.EXPECT().ExecutionComplete(ctx, f.exec.ID, execution.StatusSuccess, 200, "", `{}`, 1, 0, f.now).Return(false, tt.completeErr)
			} else {
				f.mockDB.EXPECT().ExecutionComplete(ctx, f.exec.ID, execution.StatusSuccess, 200, "", `{}`, 1, 0, f.now).Return(true, nil)
			}
			if tt.expectGet {
				f.mockDB.EXPECT().ExecutionGet(ctx, f.exec.ID).Return(nil, tt.getErr)
			}
			// PublishEvent has no expectation: the strict mock fails on any call

			before := testutil.ToFloat64(promDispatchTotal.WithLabelValues(f.s.Name, string(execution.StatusSuccess)))

			f.h.dispatch(ctx, f.s, f.exec)

			after := testutil.ToFloat64(promDispatchTotal.WithLabelValues(f.s.Name, string(execution.StatusSuccess)))
			if after-before != tt.expectDelta {
				t.Errorf("Wrong match. expect delta: %v, got: %v", tt.expectDelta, after-before)
			}
		})
	}
}

func Test_ExecuteManual_wrapped_errors(t *testing.T) {
	now := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)
	errBoom := stderrors.New("synthetic failure")

	tests := []struct {
		name string

		getErr     error
		lockErr    error
		runningErr error
		claimErr   error

		expectUnlock int
		expectText   string
	}{
		{
			name: "schedule get error",

			getErr: errBoom,

			expectUnlock: 0,
			expectText:   "could not get the schedule",
		},
		{
			name: "lock error other than busy",

			lockErr: errBoom,

			expectUnlock: 0,
			expectText:   "could not acquire the schedule lock",
		},
		{
			name: "running check error",

			runningErr: errBoom,

			expectUnlock: 1,
			expectText:   "could not check the running execution",
		},
		{
			name: "create error",

			claimErr: errBoom,

			expectUnlock: 1,
			expectText:   "could not create the execution",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			id := uuid.Must(uuid.NewV4())
			s := &schedule.Schedule{Identity: commonidentity.Identity{ID: id}, Name: "test-manual-wrapped"}

			unlockCount := 0
			unlock := func() { unlockCount++ }

			if tt.getErr != nil {
				mockDB.EXPECT().ScheduleGet(ctx, id).Return(nil, tt.getErr)
			} else {
				mockDB.EXPECT().ScheduleGet(ctx, id).Return(s, nil)
				if tt.lockErr != nil {
					mockCache.EXPECT().LockSchedule(ctx, id, scheduleLockTTL).Return(nil, tt.lockErr)
				} else {
					mockCache.EXPECT().LockSchedule(ctx, id, scheduleLockTTL).Return(unlock, nil)
					mockDB.EXPECT().ExecutionHasRunning(ctx, id).Return(false, tt.runningErr)
				}
			}
			if tt.claimErr != nil {
				mockUtil.EXPECT().TimeNow().Return(&now)
				mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, s, time.Time{}, now, execution.TriggerTypeManual, now).Return(nil, tt.claimErr)
			}

			res, err := h.ExecuteManual(ctx, id)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errBoom) {
				t.Errorf("Wrong match. expect: the original error reachable, got: %v", err)
			}
			if got := err.Error(); len(got) < len(tt.expectText) || got[:len(tt.expectText)] != tt.expectText {
				t.Errorf("Wrong match. expect prefix: %s, got: %s", tt.expectText, got)
			}
			if unlockCount != tt.expectUnlock {
				t.Errorf("Wrong match. expect unlock calls: %d, got: %d", tt.expectUnlock, unlockCount)
			}
		})
	}
}
