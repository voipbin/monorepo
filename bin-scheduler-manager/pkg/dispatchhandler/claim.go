package dispatchhandler

import (
	"context"
	stderrors "errors"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
	"monorepo/bin-scheduler-manager/pkg/cachehandler"
)

// claimAndDispatch claims the schedule's due slot and dispatches it
// (design §5.3):
//  1. redis try-once lock (contention filter — busy means another replica
//     is on it, skip silently)
//  2. overlap guard (Forbid semantics — a schedule never runs concurrently
//     with itself; the slot is late, not lost)
//  3. + 4. CAS claim + execution row in one DB transaction
//
// The lock guards only the claim critical section — it is released right
// after the claim, never held across the RPC.
func (h *dispatchHandler) claimAndDispatch(ctx context.Context, s *schedule.Schedule) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "claimAndDispatch",
		"schedule_id":   s.ID,
		"schedule_name": s.Name,
	})

	if s.TMNextRun == nil {
		// defensive: ScheduleListDue never returns such a row
		return
	}

	unlock, err := h.cache.LockSchedule(ctx, s.ID, scheduleLockTTL)
	if err != nil {
		if stderrors.Is(err, cachehandler.ErrLockBusy) {
			promDispatchTotal.WithLabelValues(s.Name, resultSkippedLock).Inc()
			return
		}
		log.Errorf("Could not acquire the schedule lock. err: %v", err)
		return
	}

	running, err := h.db.ExecutionHasRunning(ctx, s.ID)
	if err != nil {
		log.Errorf("Could not check the running execution. err: %v", err)
		unlock()
		return
	}
	if running {
		h.countOverlapSkip(s)
		unlock()
		return
	}

	now := h.utilHandler.TimeNow()

	next, err := schedule.NextRun(s.Cron, *now)
	if err != nil {
		log.Errorf("Could not compute the next run. cron: %s, err: %v", s.Cron, err)
		unlock()
		return
	}

	exec, err := h.db.ScheduleClaimAndCreateExecution(ctx, s, next, *now, execution.TriggerTypeCron, *s.TMNextRun)
	unlock() // the lock guards the claim only, never the RPC (design §5.3)
	if err != nil {
		log.Errorf("Could not claim the schedule slot. err: %v", err)
		return
	}
	if exec == nil {
		// lost the CAS to another replica (or the schedule was just
		// disabled/deleted) — not an error
		promDispatchTotal.WithLabelValues(s.Name, resultSkippedCAS).Inc()
		return
	}

	h.clearOverlapSkip(s.ID)
	// the slot is claimed: the schedule leaves the due scan, so reset the lag
	// gauge here — otherwise the series stays pinned at the last overdue value
	promScheduleLagSeconds.WithLabelValues(s.Name).Set(0)
	h.dispatch(ctx, s, exec)
}

// countOverlapSkip increments skipped_overlap once per skipped slot: repeats
// for the same tm_scheduled while the same run is in flight are suppressed
// (design §5.7).
func (h *dispatchHandler) countOverlapSkip(s *schedule.Schedule) {
	if s.TMNextRun == nil {
		return
	}

	h.muOverlap.Lock()
	defer h.muOverlap.Unlock()

	if last, ok := h.lastOverlapSkip[s.ID]; ok && last.Equal(*s.TMNextRun) {
		return
	}

	h.lastOverlapSkip[s.ID] = *s.TMNextRun
	promDispatchTotal.WithLabelValues(s.Name, resultSkippedOverlap).Inc()
}

// clearOverlapSkip resets the schedule's per-slot overlap-skip marker after a
// successful claim.
func (h *dispatchHandler) clearOverlapSkip(id uuid.UUID) {
	h.muOverlap.Lock()
	defer h.muOverlap.Unlock()

	delete(h.lastOverlapSkip, id)
}
