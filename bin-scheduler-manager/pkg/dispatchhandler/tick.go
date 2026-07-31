package dispatchhandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-scheduler-manager/models/execution"
	"monorepo/bin-scheduler-manager/models/schedule"
)

// Run runs the dispatch tick loop until the context is cancelled (design §5.2).
// Every replica runs the same loop — there is no standing leader election;
// locks are per-claim.
func (h *dispatchHandler) Run(ctx context.Context) {
	log := logrus.WithField("func", "Run")
	log.Debugf("Starting the dispatch loop. tick_interval: %s", h.tickInterval)

	ticker := time.NewTicker(h.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Debug("Stopping the dispatch loop.")
			return
		case <-ticker.C:
			h.tick(ctx)
		}
	}
}

// tick executes one dispatch scan (design §5.2):
// reap abandoned -> refresh gauges -> init uninitialized -> claim due slots.
func (h *dispatchHandler) tick(ctx context.Context) {
	now := h.utilHandler.TimeNow()

	h.reapAbandoned(ctx, *now)
	h.refreshExecutionRows(ctx)
	h.initUninitialized(ctx, *now)
	h.dispatchDue(ctx, *now)
}

// reapAbandoned marks running executions past their tm_deadline as abandoned
// (design §5.6).
func (h *dispatchHandler) reapAbandoned(ctx context.Context, now time.Time) {
	log := logrus.WithField("func", "reapAbandoned")

	count, err := h.db.ExecutionReapAbandoned(ctx, now)
	if err != nil {
		log.Errorf("Could not reap abandoned executions. err: %v", err)
		return
	}

	if count > 0 {
		// The reap is a bulk statement without per-schedule granularity, so
		// the abandoned result is counted without a schedule_name label
		// (empty label — a deliberate deviation from the per-schedule labels
		// of the other results, documented on the metric help text).
		promDispatchTotal.WithLabelValues("", string(execution.StatusAbandoned)).Add(float64(count))
		log.Debugf("Reaped abandoned executions. count: %d", count)
	}
}

// refreshExecutionRows refreshes the execution_rows gauge once per tick —
// never per Prometheus scrape (design §5.7).
func (h *dispatchHandler) refreshExecutionRows(ctx context.Context) {
	log := logrus.WithField("func", "refreshExecutionRows")

	count, err := h.db.ExecutionCountAll(ctx)
	if err != nil {
		log.Errorf("Could not count the executions. err: %v", err)
		return
	}

	promExecutionRows.Set(float64(count))
}

// initUninitialized computes tm_next_run for enabled schedules whose
// tm_next_run is NULL (fresh create, cron change, or re-enable), forward from
// now — a long-disabled schedule never catch-up-fires (design §5.2).
func (h *dispatchHandler) initUninitialized(ctx context.Context, now time.Time) {
	log := logrus.WithField("func", "initUninitialized")

	schedules, err := h.db.ScheduleListUninitialized(ctx, scanLimit)
	if err != nil {
		log.Errorf("Could not list uninitialized schedules. err: %v", err)
		return
	}

	for _, s := range schedules {
		next, errNext := schedule.NextRun(s.Cron, now)
		if errNext != nil {
			log.Errorf("Could not compute the next run. schedule_id: %s, cron: %s, err: %v", s.ID, s.Cron, errNext)
			continue
		}

		if _, errInit := h.db.ScheduleInitNextRun(ctx, s.ID, next); errInit != nil {
			log.Errorf("Could not init the next run. schedule_id: %s, err: %v", s.ID, errInit)
			continue
		}
	}
}

// dispatchDue scans due schedules and claims each in a semaphore-bounded
// goroutine (design §5.2).
func (h *dispatchHandler) dispatchDue(ctx context.Context, now time.Time) {
	log := logrus.WithField("func", "dispatchDue")

	schedules, err := h.db.ScheduleListDue(ctx, now, scanLimit)
	if err != nil {
		log.Errorf("Could not list due schedules. err: %v", err)
		return
	}

	for _, s := range schedules {
		if s.TMNextRun != nil {
			if lag := now.Sub(*s.TMNextRun).Seconds(); lag > 0 {
				promScheduleLagSeconds.WithLabelValues(s.Name).Set(lag)
			}
		}

		select {
		case h.sem <- struct{}{}:
		case <-ctx.Done():
			return
		}

		go func(target *schedule.Schedule) {
			defer func() { <-h.sem }()
			h.claimAndDispatch(ctx, target)
		}(s)
	}
}
