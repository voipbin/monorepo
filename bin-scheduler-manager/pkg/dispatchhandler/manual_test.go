package dispatchhandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
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

func Test_ExecuteManual_error(t *testing.T) {
	deletedTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)

	tests := []struct {
		name string

		scheduleID uuid.UUID

		responseSchedule    *schedule.Schedule
		responseScheduleErr error
		responseLockErr     error
		responseRunning     bool
		expectClaim         bool

		expectStatus cerrors.Status
	}{
		{
			name: "not found",

			scheduleID: uuid.FromStringOrNil("718293a4-1a12-11ef-8f0f-6f2a2b7f9d3e"),

			responseScheduleErr: dbhandler.ErrNotFound,

			expectStatus: cerrors.StatusNotFound,
		},
		{
			name: "deleted schedule",

			scheduleID: uuid.FromStringOrNil("8293a4b5-1a12-11ef-9d3e-6f2a2b7f9d3e"),

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8293a4b5-1a12-11ef-9d3e-6f2a2b7f9d3e"),
				},
				TMDelete: &deletedTime,
			},

			expectStatus: cerrors.StatusNotFound,
		},
		{
			name: "lock busy",

			scheduleID: uuid.FromStringOrNil("93a4b5c6-1a12-11ef-a06f-6f2a2b7f9d3e"),

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93a4b5c6-1a12-11ef-a06f-6f2a2b7f9d3e"),
				},
			},
			responseLockErr: cachehandler.ErrLockBusy,

			expectStatus: cerrors.StatusFailedPrecondition,
		},
		{
			name: "overlap running execution",

			scheduleID: uuid.FromStringOrNil("a4b5c6d7-1a12-11ef-b1cd-6f2a2b7f9d3e"),

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4b5c6d7-1a12-11ef-b1cd-6f2a2b7f9d3e"),
				},
			},
			responseRunning: true,

			expectStatus: cerrors.StatusFailedPrecondition,
		},
		{
			name: "concurrent manual fire deduped",

			scheduleID: uuid.FromStringOrNil("b5c6d7e8-1a12-11ef-8ff5-6f2a2b7f9d3e"),

			responseSchedule: &schedule.Schedule{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5c6d7e8-1a12-11ef-8ff5-6f2a2b7f9d3e"),
				},
			},
			expectClaim: true,

			expectStatus: cerrors.StatusFailedPrecondition,
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
				db:              mockDB,
				cache:           mockCache,
				lastOverlapSkip: map[uuid.UUID]time.Time{},
			}

			ctx := context.Background()

			mockDB.EXPECT().ScheduleGet(ctx, tt.scheduleID).Return(tt.responseSchedule, tt.responseScheduleErr)

			if tt.responseScheduleErr == nil && tt.responseSchedule.TMDelete == nil {
				if tt.responseLockErr != nil {
					mockCache.EXPECT().LockSchedule(ctx, tt.scheduleID, scheduleLockTTL).Return(nil, tt.responseLockErr)
				} else {
					unlock := func() {}
					mockCache.EXPECT().LockSchedule(ctx, tt.scheduleID, scheduleLockTTL).Return(unlock, nil)
					mockDB.EXPECT().ExecutionHasRunning(ctx, tt.scheduleID).Return(tt.responseRunning, nil)
					if tt.expectClaim {
						mockUtil.EXPECT().TimeNow().Return(&now)
						mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, tt.responseSchedule, time.Time{}, now, execution.TriggerTypeManual, now).Return(nil, nil)
					}
				}
			}

			_, err := h.ExecuteManual(ctx, tt.scheduleID)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
				return
			}

			var vErr *cerrors.VoipbinError
			if !stderrors.As(err, &vErr) {
				t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
				return
			}
			if vErr.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectStatus, vErr.Status)
			}
		})
	}
}

func Test_ExecuteManual_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &dispatchHandler{
		utilHandler:     mockUtil,
		reqHandler:      mockReq,
		db:              mockDB,
		cache:           mockCache,
		notifyHandler:   mockNotify,
		retryBackoff:    time.Millisecond,
		lastOverlapSkip: map[uuid.UUID]time.Time{},
	}

	ctx := context.Background()

	now := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)

	// disabled schedule — a manual test-fire works on disabled schedules
	s := &schedule.Schedule{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("c6d7e8f9-1a12-11ef-9be0-6f2a2b7f9d3e"),
		},
		Name: "test-manual-success",

		TargetQueue:    "bin-manager.scheduler-manager.request",
		TargetURI:      "/v1/backups",
		TargetMethod:   "POST",
		TargetDataType: "application/json",
		TargetData:     []byte(`{}`),

		TimeoutMS: 30000,
		RetryMax:  0,
		Enabled:   false,
	}

	exec := &execution.Execution{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("d7e8f90a-1a12-11ef-a3c3-6f2a2b7f9d3e"),
		},
		ScheduleID:  s.ID,
		TriggerType: execution.TriggerTypeManual,
		Status:      execution.StatusRunning,
	}

	unlock := func() {}
	mockDB.EXPECT().ScheduleGet(ctx, s.ID).Return(s, nil)
	mockCache.EXPECT().LockSchedule(ctx, s.ID, scheduleLockTTL).Return(unlock, nil)
	mockDB.EXPECT().ExecutionHasRunning(ctx, s.ID).Return(false, nil)
	mockUtil.EXPECT().TimeNow().Return(&now).AnyTimes()
	mockDB.EXPECT().ScheduleClaimAndCreateExecution(ctx, s, time.Time{}, now, execution.TriggerTypeManual, now).Return(exec, nil)

	// the async dispatch runs on a detached context — match any context
	done := make(chan struct{})
	mockReq.EXPECT().SendRequest(gomock.Any(), gomock.Any(), s.TargetURI, sock.RequestMethodPost, s.TimeoutMS, 0, s.TargetDataType, gomock.Any()).Return(&sock.Response{StatusCode: 200, Data: []byte(`{"path":"/backups/voipbin.sql.gz","bytes":100}`)}, nil)
	mockDB.EXPECT().ExecutionComplete(gomock.Any(), exec.ID, execution.StatusSuccess, 200, "", `{"path":"/backups/voipbin.sql.gz","bytes":100}`, 1, 0, now).Return(true, nil)
	mockDB.EXPECT().ExecutionGet(gomock.Any(), exec.ID).Return(exec, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), schedule.EventTypeExecutionSucceeded, exec).Do(func(_ context.Context, _ string, _ any) {
		close(done)
	})

	res, err := h.ExecuteManual(ctx, s.ID)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(res, exec) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", exec, res)
	}

	// wait for the async dispatch to finish before the controller verifies
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("Wrong match. expect: async dispatch completed, got: timeout")
	}
}
