package casehandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// maxInsertRetries bounds the ON DUPLICATE KEY retry loop on the insert
// branches (design §4.2's round-2 correction: retry, don't assume the
// first re-select is final). Loop exhaustion is an extremely rare
// thundering-herd scenario; see design §4.2's "Loop exhaustion" note.
const maxInsertRetries = 3

// ErrGetOrCreateExhausted is returned when all maxInsertRetries attempts
// to insert-or-reselect an open Case collide (design §4.2's "Loop
// exhaustion" path). Callers must surface this as a transient 5xx, not
// silently drop the triggering event -- at-least-once delivery ensures a
// retry.
var ErrGetOrCreateExhausted = fmt.Errorf("could not get-or-create case: exhausted retries under sustained conflict")

// GetOrCreate implements design doc §4's get-or-create algorithm exactly.
func (h *caseHandler) GetOrCreate(
	ctx context.Context,
	customerID uuid.UUID,
	peerType commonaddress.Type,
	peerTarget, referenceType string,
	caseIDHint *uuid.UUID,
) (*kase.Case, error) {
	now := h.utilHandler.TimeNow()

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction. GetOrCreate. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, isNewCase, err := h.getOrCreateInTx(ctx, tx, customerID, peerType, peerTarget, referenceType, caseIDHint, now)
	if err != nil {
		return nil, err
	}

	if err := h.db.CaseUpdateTMUpdateTx(ctx, tx, res.ID, now); err != nil {
		return nil, fmt.Errorf("could not bump tm_update. GetOrCreate. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction. GetOrCreate. err: %v", err)
	}
	committed = true

	_ = isNewCase // consumed by later sub-tasks (contact auto-match, §4.4 proactive link)

	res.TMUpdate = now
	return res, nil
}

// getOrCreateInTx runs design §4 steps 1 (case resolution) inside tx.
// Returns the resolved Case and whether it was freshly inserted this call
// (vs. reused/hint-matched).
func (h *caseHandler) getOrCreateInTx(
	ctx context.Context,
	tx *sql.Tx,
	customerID uuid.UUID,
	peerType commonaddress.Type,
	peerTarget, referenceType string,
	caseIDHint *uuid.UUID,
	now *time.Time,
) (*kase.Case, bool, error) {
	// Step 1a: explicit case_id hint (design §4.3). Validated: correct
	// tenant, still open. An invalid/stale/closed hint is never an error --
	// it just falls through to the peer/reference_type path as if no hint
	// were given.
	if caseIDHint != nil {
		hinted, err := h.db.CaseGetByIDForUpdate(ctx, tx, customerID, *caseIDHint)
		if err != nil {
			return nil, false, fmt.Errorf("could not validate case_id hint. GetOrCreate. err: %v", err)
		}
		if hinted != nil {
			return hinted, false, nil
		}
		// fall through
	}

	// Step 1b: peer/reference_type resolution.
	found, err := h.db.CaseGetOpenByPeer(ctx, tx, customerID, peerType, peerTarget, referenceType)
	if err != nil {
		return nil, false, fmt.Errorf("could not look up open case. GetOrCreate. err: %v", err)
	}

	timeoutThreshold := time.Duration(config.Get().CaseTimeoutHours) * time.Hour

	if found != nil {
		if found.TMUpdate != nil && now.Sub(*found.TMUpdate) < timeoutThreshold {
			// Reuse.
			return found, false, nil
		}

		// Timed out: close it, then insert a fresh replacement chained via
		// previous_case_id.
		if _, err := h.db.CaseUpdateStatusClosedTx(ctx, tx, found.ID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, now); err != nil {
			return nil, false, fmt.Errorf("could not close timed-out case. GetOrCreate. err: %v", err)
		}
		return h.insertWithRetry(ctx, tx, customerID, peerType, peerTarget, referenceType, &found.ID, now)
	}

	// Step 1c: no open case at all -- fresh insert, chained to the last
	// closed case for this peer (if any) via previous_case_id.
	lastClosed, err := h.db.CaseGetLastClosedByPeerTx(ctx, tx, customerID, peerType, peerTarget, referenceType)
	if err != nil {
		return nil, false, fmt.Errorf("could not look up last closed case. GetOrCreate. err: %v", err)
	}
	var previousCaseID *uuid.UUID
	if lastClosed != nil {
		previousCaseID = &lastClosed.ID
	}
	return h.insertWithRetry(ctx, tx, customerID, peerType, peerTarget, referenceType, previousCaseID, now)
}

// insertWithRetry implements design §4.2's bounded retry loop: attempt an
// INSERT; on a uq_case_open_peer conflict, re-select the winning row WITH
// FOR UPDATE (extending this transaction's lock to it so no other
// transaction can close it out from under us before we commit) and use
// it if still open; otherwise loop and retry the insert (the row we
// raced against may itself have since closed/timed out).
func (h *caseHandler) insertWithRetry(
	ctx context.Context,
	tx *sql.Tx,
	customerID uuid.UUID,
	peerType commonaddress.Type,
	peerTarget, referenceType string,
	previousCaseID *uuid.UUID,
	now *time.Time,
) (*kase.Case, bool, error) {
	for attempt := 0; attempt < maxInsertRetries; attempt++ {
		newCase := &kase.Case{
			ID:             h.utilHandler.UUIDCreate(),
			CustomerID:     customerID,
			PeerType:       peerType,
			PeerTarget:     peerTarget,
			ReferenceType:  referenceType,
			Status:         kase.StatusOpen,
			OpenedAt:       now,
			PreviousCaseID: previousCaseID,
			TMCreate:       now,
			TMUpdate:       now,
		}

		err := h.db.CaseInsertTx(ctx, tx, newCase)
		if err == nil {
			return newCase, true, nil
		}
		if err != dbhandler.ErrDuplicate {
			return nil, false, fmt.Errorf("could not insert case. GetOrCreate. err: %v", err)
		}

		// Conflict: another transaction won. Re-select the winner, locked.
		winner, selErr := h.db.CaseGetOpenByPeer(ctx, tx, customerID, peerType, peerTarget, referenceType)
		if selErr != nil {
			return nil, false, fmt.Errorf("could not re-select after insert conflict. GetOrCreate. err: %v", selErr)
		}
		if winner != nil {
			return winner, false, nil
		}
		// The row we raced against itself transitioned out of 'open'
		// before we re-selected it (rarer second race) -- loop and retry.
	}

	return nil, false, ErrGetOrCreateExhausted
}
