package campaignhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/sirupsen/logrus"

	"monorepo/bin-campaign-manager/models/campaign"
)

// Package-level tuning knobs for ReconcileOrphanedFlows (VOIP-1444).
//
// These are declared as package-level vars, not true Go constants,
// specifically so unit tests in this package can override scanLimit and
// reconcilePassTimeout to small, deterministic values (constructing a
// scanLimit-sized fixture at the production default of 500 rows, or
// waiting out a 90s timeout, would make the test suite impractically slow
// and unwieldy). Production code never mutates these; only tests do,
// always restoring the original value via defer.
var (
	// window bounds how far back CampaignListDeletedSince looks for
	// deleted-campaign candidates. Anything deleted before this is
	// explicitly out of scope (see the design doc's "Proposed scope").
	window = 7 * 24 * time.Hour

	// scanLimit bounds the number of candidates examined in a single
	// pass, so one scheduled run can never issue an unbounded number of
	// RPCs. Must exceed the number of campaign deletions per cron
	// interval (not per window) — see "Scan order and coverage" in the
	// design/plan docs for the full safety-condition analysis.
	scanLimit = uint64(500)

	// reconcilePassTimeout bounds this method's own execution via an
	// internal context.WithTimeout — required because this service's RPC
	// listenhandler builds every request's context with
	// context.Background() (no deadline propagation), so nothing else
	// bounds a pass's duration. Must satisfy two independently
	// load-bearing constraints against the seeded bin-schedule-manager
	// schedule (see docs/operations.md and the design/plan docs' "Scan
	// order and coverage" / "self-imposed pass timeout" sections):
	//   (a) strictly below the schedule's timeout_ms, with margin for
	//       real message-delivery delay — the actual mutual-exclusion
	//       guarantee (keeps bin-schedule-manager's "Forbid overlap"
	//       guard, ExecutionHasRunning, effective for this pass's entire
	//       physical duration).
	//   (b) well below the schedule's cron interval — a cadence/liveness
	//       requirement, not a concurrency guard (keeps the schedule from
	//       chronically observing dispatch-level skipped_overlap skips).
	reconcilePassTimeout = 90 * time.Second

	// defaultRecentInterval is the conservative fallback used when the
	// caller-supplied recentIntervalSec is missing or non-positive (e.g.
	// an old/malformed dispatch). Never panics or fails the pass over a
	// malformed rate-signal input.
	defaultRecentInterval = 24 * time.Hour

	// windowEdgeMarginFraction defines the "near the window's outer edge"
	// zone for the per-candidate window-edge warning: a genuine orphan
	// whose owning campaign was deleted within this fraction of `window`
	// from the outer cutoff logs a warning, since a bin-flow-manager
	// outage lasting longer than `window` would let that leak age out of
	// the scan and silently merge into the out-of-scope historical
	// backlog.
	windowEdgeMarginFraction = 0.10
)

// ReconcileOrphanedFlows scans campaigns deleted within the last `window`
// (bounded to at most scanLimit rows, most-recently-deleted first) and
// deletes the backing flow of any campaign whose flow is still live —
// closing the gap left by Delete()'s best-effort FlowV1FlowDelete call
// (VOIP-1443) failing silently. See
// docs/plans/2026-09-02-voip-1444-orphaned-flow-reconciliation-plan.md and
// its paired design doc for the full analysis and round-by-round review
// history.
//
// recentIntervalSec is the caller-supplied cron interval, in seconds, of
// the bin-schedule-manager schedule dispatching this pass —
// bin-campaign-manager has no way to read bin-schedule-manager's own cron
// field, so it must be threaded through explicitly on every request. A
// missing or non-positive value falls back to a conservative 24h default
// (logged as a warning), never panics or fails the pass.
//
// err is non-nil only when the initial CampaignListDeletedSince query
// itself fails — the pass could not run at all. Every other outcome
// (per-row failures, a self-timeout bail-out) is counted in the returned
// ReconcileResult, never propagated as err; HTTP status at the
// listenhandler layer is 200 in all of those cases.
func (h *campaignHandler) ReconcileOrphanedFlows(ctx context.Context, recentIntervalSec int64) (campaign.ReconcileResult, error) {
	result := campaign.ReconcileResult{}

	log := logrus.WithFields(logrus.Fields{
		"func": "ReconcileOrphanedFlows",
	})

	passStartPtr := h.util.TimeNow()
	passStart := *passStartPtr

	// Self-imposed pass timeout — see the reconcilePassTimeout doc-comment
	// above for why this is required and what it must satisfy.
	pctx, cancel := context.WithTimeout(ctx, reconcilePassTimeout)
	defer cancel()

	since := passStart.Add(-window)
	candidates, err := h.db.CampaignListDeletedSince(pctx, since, scanLimit)
	if err != nil {
		return result, fmt.Errorf("could not list deleted campaigns. ReconcileOrphanedFlows. err: %v", err)
	}

	// Compute Saturated and RecentSaturated in a dedicated, RPC-free pass
	// over the fetched slice, before any FlowV1FlowGet/FlowV1FlowDelete
	// call — this must not be interleaved with the RPC loop below, since
	// the self-imposed pass timeout could otherwise cut the count short
	// of scanLimit on a genuinely saturated batch, producing a false
	// negative in exactly the overload case this signal exists to catch.
	result.Saturated = uint64(len(candidates)) == scanLimit

	recentInterval := time.Duration(recentIntervalSec) * time.Second
	if recentIntervalSec <= 0 {
		log.Warnf("recentIntervalSec is missing or non-positive (%d); falling back to default %s", recentIntervalSec, defaultRecentInterval)
		recentInterval = defaultRecentInterval
	}
	recentCutoff := passStart.Add(-recentInterval)

	recentCount := 0
	for _, c := range candidates {
		if c.TMDelete != nil && !c.TMDelete.Before(recentCutoff) {
			recentCount++
		}
	}
	result.RecentSaturated = uint64(recentCount) == scanLimit

	windowEdgeThreshold := since.Add(time.Duration(float64(window) * windowEdgeMarginFraction))

	// RPC loop proper.
	for _, c := range candidates {
		if pctx.Err() != nil {
			result.Partial = true
			log.Warn("self-imposed pass timeout reached mid-pass; returning partial results")
			return result, nil
		}

		rowLog := log.WithFields(logrus.Fields{
			"campaign_id": c.ID,
			"flow_id":     c.FlowID,
		})

		// Defensive, second layer of defense: CampaignListDeletedSince
		// already filters on tm_delete >= since (which excludes live
		// campaigns via NULL-comparison semantics), but a future query
		// regression accidentally including live campaigns should be
		// loud, not silent, given this job mutates data across all
		// customers.
		if c.TMDelete == nil {
			rowLog.Warn("candidate campaign has nil TMDelete; skipping (defensive guard against a query regression)")
			result.Skipped++
			continue
		}

		f, errGet := h.reqHandler.FlowV1FlowGet(pctx, c.FlowID)
		if errGet != nil {
			var ve *cerrors.VoipbinError
			if stderrors.As(errGet, &ve) && ve.Status == cerrors.StatusNotFound {
				// The flow row genuinely doesn't exist -- a legitimately
				// clean state, same bucket as an already-deleted flow.
				result.Skipped++
				continue
			}

			// Any other error (RPC timeout, backend error, an open
			// circuit breaker) is a failure, not a clean state. A
			// bin-flow-manager outage fails every remaining candidate on
			// this exact branch, so it must be observable via the shared
			// counter, not just the in-memory count.
			rowLog.Errorf("could not get flow. err: %v", errGet)
			promCampaignFlowReconcileFailedTotal.Inc()
			result.Failed++
			continue
		}

		if f.TMDelete != nil {
			// Already cleaned up (by an earlier pass, or a successful
			// Delete() call) -- not an orphan, do not re-delete.
			result.Skipped++
			continue
		}

		// Genuinely orphaned: owning campaign is deleted, flow is live.
		if c.TMDelete.Before(windowEdgeThreshold) {
			rowLog.WithField("tm_delete", c.TMDelete).Warn("orphan's owning campaign was deleted near the reconciliation window's outer edge; window may need widening")
		}

		if _, errDelete := h.reqHandler.FlowV1FlowDelete(pctx, c.FlowID); errDelete != nil {
			rowLog.Errorf("could not delete orphaned flow. err: %v", errDelete)
			promCampaignFlowReconcileFailedTotal.Inc()
			result.Failed++
			continue
		}

		result.Cleaned++
		promCampaignFlowReconcileCleanedTotal.Inc()
	}

	return result, nil
}
