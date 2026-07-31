package dispatchhandler

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
	"monorepo/bin-scheduler-manager/pkg/dbhandler"
)

// ExecuteManual fires the schedule immediately (design §6):
//   - never touches tm_next_run/tm_last_run (a test-fire must not consume the
//     next cron slot) — the dbhandler skips the schedule mutation for the
//     manual trigger type
//   - works on disabled schedules (that is the point of a manual test-fire)
//   - refused with a 409-mappable FailedPrecondition while any run of the
//     schedule is in flight (same lock + overlap guard as the cron path)
//   - dispatches asynchronously and returns the created running execution —
//     the RPC caller must not block for the run's duration (e.g. a 30-minute
//     backup)
func (h *dispatchHandler) ExecuteManual(ctx context.Context, scheduleID uuid.UUID) (*execution.Execution, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "ExecuteManual",
		"schedule_id": scheduleID,
	})
	log.Debug("Executing the schedule manually.")

	s, err := h.db.ScheduleGet(ctx, scheduleID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_NOT_FOUND",
				"The schedule was not found.",
			).Wrap(err)
		}
		return nil, errors.Wrap(err, "could not get the schedule")
	}
	if s.TMDelete != nil {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameSchedulerManager,
			"SCHEDULE_NOT_FOUND",
			"The schedule was deleted.",
		)
	}

	unlock, err := h.cache.LockSchedule(ctx, scheduleID, scheduleLockTTL)
	if err != nil {
		if stderrors.Is(err, cachehandler.ErrLockBusy) {
			return nil, cerrors.FailedPrecondition(
				commonoutline.ServiceNameSchedulerManager,
				"SCHEDULE_BUSY",
				"The schedule is being dispatched. Try again later.",
			).Wrap(err)
		}
		return nil, errors.Wrap(err, "could not acquire the schedule lock")
	}

	running, err := h.db.ExecutionHasRunning(ctx, scheduleID)
	if err != nil {
		unlock()
		return nil, errors.Wrap(err, "could not check the running execution")
	}
	if running {
		unlock()
		return nil, cerrors.FailedPrecondition(
			commonoutline.ServiceNameSchedulerManager,
			"EXECUTION_IN_PROGRESS",
			"The schedule already has a running execution.",
		)
	}

	now := h.utilHandler.TimeNow()

	// the next argument is ignored for the manual trigger type (the schedule
	// row is not mutated); tm_scheduled is the claim wall time (design §6)
	exec, err := h.db.ScheduleClaimAndCreateExecution(ctx, s, time.Time{}, *now, execution.TriggerTypeManual, *now)
	unlock()
	if err != nil {
		return nil, errors.Wrap(err, "could not create the execution")
	}
	if exec == nil {
		// a concurrent manual fire was deduped by the unique-key backstop
		return nil, cerrors.FailedPrecondition(
			commonoutline.ServiceNameSchedulerManager,
			"EXECUTION_IN_PROGRESS",
			"The schedule already has a running execution.",
		)
	}

	// dispatch asynchronously: the dispatch outlives the RPC context, so it
	// runs on a context detached from the caller's cancellation
	go h.dispatch(context.WithoutCancel(ctx), s, exec)

	return exec, nil
}
