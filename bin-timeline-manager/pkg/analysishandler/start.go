package analysishandler

import (
	"context"
	"errors"
	"fmt"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
)

// Start triggers (or returns an in-flight/existing) analysis for an ended
// activeflow. See design §5.2. Ownership is checked FIRST, then the ended gate,
// so a customer never receives the distinct "not-ended" signal for a foreign
// activeflow (review F3).
func (h *analysisHandler) Start(ctx context.Context, customerID, activeflowID uuid.UUID, reanalyze bool) (*analysis.Analysis, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Start",
		"customer_id":   customerID,
		"activeflow_id": activeflowID,
		"reanalyze":     reanalyze,
	})

	// 1) ownership FIRST, then ended gate.
	af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, activeflowID)
	if err != nil {
		// absent activeflow -> masked not-found (no existence oracle).
		return nil, ErrNotFound
	}
	if af.CustomerID != customerID {
		// foreign activeflow -> masked not-found (no existence/ownership oracle).
		return nil, ErrNotFound
	}
	if af.Status != fmactiveflow.StatusEnded {
		return nil, ErrActiveflowNotEnded
	}

	// 2) existing-record policy.
	existing, err := h.dbHandler.AnalysisGetByActiveflowID(ctx, customerID, activeflowID)
	switch {
	case err == nil:
		res, handled, perr := h.applyExistingPolicy(ctx, existing, reanalyze)
		if perr != nil {
			return nil, perr
		}
		if handled {
			return res, nil
		}
		// policy decided to (re)start; res is the row to run the chain on.
		h.kickoff(res.ID, customerID, activeflowID)
		return res, nil

	case errors.Is(err, analysisdbhandler.ErrNotFound):
		// no record -> a genuinely new analysis. Enforce the per-customer
		// concurrency cap BEFORE spending on a new chain (design F1).
		count, cerr := h.dbHandler.AnalysisCountProgressing(ctx, customerID)
		if cerr != nil {
			return nil, fmt.Errorf("Start: could not count progressing analyses. err: %v", cerr)
		}
		if count >= analysisMaxProgressingPerCustomer {
			return nil, ErrConcurrencyLimit
		}

		// create progressing, handle the concurrent-create race.
		created, createErr := h.createProgressing(ctx, customerID, activeflowID)
		if createErr != nil {
			return nil, createErr
		}
		// only kick the chain when WE created the row (not when we returned a
		// concurrent in-flight one).
		if created.justCreated {
			h.kickoff(created.row.ID, customerID, activeflowID)
		}
		return created.row, nil

	default:
		log.Errorf("could not look up existing analysis. err: %v", err)
		return nil, fmt.Errorf("Start: could not look up existing analysis. err: %v", err)
	}
}

// applyExistingPolicy implements the existing-record decision table (design §5.2).
// Returns (row, handled=true) when the caller should return immediately without
// starting a chain; (row, handled=false) when the caller should start the chain.
func (h *analysisHandler) applyExistingPolicy(ctx context.Context, existing *analysis.Analysis, reanalyze bool) (*analysis.Analysis, bool, error) {
	switch existing.Status {
	case analysis.StatusProgressing:
		// a run is in flight; never double-spend, even with reanalyze=true.
		return existing, true, nil

	case analysis.StatusCompleted, analysis.StatusFailed:
		if !reanalyze {
			// idempotent return of the existing terminal record.
			return existing, true, nil
		}
		// reanalyze requested: cooldown gate first.
		if existing.TMUpdate != nil {
			elapsed := h.utilHandler.TimeNow().Sub(*existing.TMUpdate)
			if elapsed < analysisReanalyzeCooldown {
				return nil, true, ErrReanalyzeCooldown
			}
		}
		// reset in place (CAS). 0 rows means another reanalyze already won.
		n, err := h.dbHandler.AnalysisReset(ctx, existing.ID)
		if err != nil {
			return nil, true, fmt.Errorf("applyExistingPolicy: could not reset for reanalyze. err: %v", err)
		}
		if n == 0 {
			// lost the race; return the in-flight row (no second goroutine).
			inflight, gerr := h.dbHandler.AnalysisGet(ctx, existing.CustomerID, existing.ID)
			if gerr != nil {
				return nil, true, fmt.Errorf("applyExistingPolicy: could not re-read in-flight row. err: %v", gerr)
			}
			return inflight, true, nil
		}
		// we won the reset; re-read the now-progressing row and start the chain.
		reset, gerr := h.dbHandler.AnalysisGet(ctx, existing.CustomerID, existing.ID)
		if gerr != nil {
			return nil, true, fmt.Errorf("applyExistingPolicy: could not re-read reset row. err: %v", gerr)
		}
		return reset, false, nil

	default:
		// unknown status — treat defensively as in-flight (do not spend).
		return existing, true, nil
	}
}

type createResult struct {
	row         *analysis.Analysis
	justCreated bool
}

// createProgressing inserts a fresh progressing row, made race-safe by the
// UNIQUE(activeflow_id) constraint: on a duplicate-key the concurrent in-flight
// row is re-read and returned instead of surfacing an error (review H3).
func (h *analysisHandler) createProgressing(ctx context.Context, customerID, activeflowID uuid.UUID) (*createResult, error) {
	row := &analysis.Analysis{
		Identity: commonIdentity(h.utilHandler.UUIDCreate(), customerID),
	}
	row.ActiveflowID = activeflowID
	row.Status = analysis.StatusProgressing

	err := h.dbHandler.AnalysisCreate(ctx, row)
	switch {
	case err == nil:
		return &createResult{row: row, justCreated: true}, nil

	case errors.Is(err, analysisdbhandler.ErrDuplicate):
		inflight, gerr := h.dbHandler.AnalysisGetByActiveflowID(ctx, customerID, activeflowID)
		if gerr != nil {
			return nil, fmt.Errorf("createProgressing: could not re-read in-flight row after dup. err: %v", gerr)
		}
		return &createResult{row: inflight, justCreated: false}, nil

	default:
		return nil, fmt.Errorf("createProgressing: could not create progressing row. err: %v", err)
	}
}
