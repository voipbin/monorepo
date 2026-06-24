package analysishandler

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-timeline-manager/models/analysis"
)

// kickoff launches the async analysis chain for a progressing row. It is
// bounded by a semaphore, recovers from panics, and persists the terminal
// result on a fresh context so the row never stays stuck in progressing. The
// activeflow fetched in Start is threaded in (not re-Got) so enrichment keys off
// the same snapshot (#10).
func (h *analysisHandler) kickoff(analysisID, customerID, activeflowID uuid.UUID, af *fmactiveflow.Activeflow) {
	h.metricStarted.Inc()

	go func() {
		start := time.Now()
		log := logrus.WithFields(logrus.Fields{
			"func":          "kickoff",
			"analysis_id":   analysisID,
			"customer_id":   customerID,
			"activeflow_id": activeflowID,
		})

		// bound concurrency.
		h.sem <- struct{}{}
		defer func() { <-h.sem }()

		// recover so a panic in the chain becomes a failed record, not a crash.
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("recovered from panic in analysis chain. r: %v, stack: %s", r, debug.Stack())
				h.persistFailed(analysisID, "internal error during analysis")
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), analysisJobTimeout)
		defer cancel()

		verdictJSON, modelUsed, err := h.runChain(ctx, customerID, activeflowID, af)
		if err != nil {
			log.Errorf("analysis chain failed. err: %v", err)
			// sanitized, operator-safe message (no raw provider errors/stacks — review L2).
			h.persistFailed(analysisID, "analysis failed to produce a result")
			h.metricCompleted.WithLabelValues(string(analysis.StatusFailed)).Inc()
			h.metricDuration.Observe(time.Since(start).Seconds())
			return
		}

		h.persistCompleted(analysisID, verdictJSON, modelUsed)
		h.metricCompleted.WithLabelValues(string(analysis.StatusCompleted)).Inc()
		h.metricDuration.Observe(time.Since(start).Seconds())
		log.WithField("duration", time.Since(start)).Debug("analysis chain completed.")
	}()
}

// persistCompleted writes the terminal completed result on a fresh short-lived
// context so it lands even if the job context expired (review #8 guard lives in
// the dbhandler UPDATE: only a still-progressing, non-deleted row is updated).
func (h *analysisHandler) persistCompleted(analysisID uuid.UUID, verdictJSON []byte, modelUsed string) {
	ctx, cancel := context.WithTimeout(context.Background(), analysisFinalWriteTimeout)
	defer cancel()

	n, err := h.dbHandler.AnalysisUpdateResult(ctx, analysisID, analysis.StatusCompleted, verdictJSON, modelUsed, "")
	if err != nil {
		logrus.WithField("analysis_id", analysisID).Errorf("could not persist completed result. err: %v", err)
		return
	}
	if n == 0 {
		logrus.WithField("analysis_id", analysisID).Info("completed result not persisted (row deleted or superseded mid-run).")
	}
}

// persistFailed writes the terminal failed status with a sanitized message.
func (h *analysisHandler) persistFailed(analysisID uuid.UUID, sanitizedMsg string) {
	ctx, cancel := context.WithTimeout(context.Background(), analysisFinalWriteTimeout)
	defer cancel()

	n, err := h.dbHandler.AnalysisUpdateResult(ctx, analysisID, analysis.StatusFailed, nil, "", sanitizedMsg)
	if err != nil {
		logrus.WithField("analysis_id", analysisID).Errorf("could not persist failed status. err: %v", err)
		return
	}
	if n == 0 {
		logrus.WithField("analysis_id", analysisID).Info("failed status not persisted (row deleted or superseded mid-run).")
	}
}
