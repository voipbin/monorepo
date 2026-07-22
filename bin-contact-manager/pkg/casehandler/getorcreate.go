package casehandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/internal/config"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
)

// maxInsertRetries bounds the ON DUPLICATE KEY retry loop on the insert
// branches (design §4.2's round-2 correction: retry, don't assume the
// first re-select is final). Loop exhaustion is an extremely rare
// thundering-herd scenario; see design §4.2's "Loop exhaustion" note.
const maxInsertRetries = 3

// maxDeadlockRetries bounds the outer retry loop that restarts the ENTIRE
// GetOrCreate transaction (fresh BeginTx) when the underlying driver
// reports a MySQL deadlock (errno 1213, VOIP-1232). This is a distinct,
// wider mechanism than maxInsertRetries above: an InnoDB deadlock kills
// the whole transaction server-side (unlike a plain uq_case_open_peer
// conflict, which is safely retryable within the SAME tx via a locked
// re-select), so recovery here must restart the transaction from
// scratch, not just retry one statement.
const maxDeadlockRetries = 3

// ErrGetOrCreateExhausted is returned when all maxInsertRetries attempts
// to insert-or-reselect an open Case collide (design §4.2's "Loop
// exhaustion" path). Callers must surface this as a transient 5xx, not
// silently drop the triggering event -- at-least-once delivery ensures a
// retry.
var ErrGetOrCreateExhausted = fmt.Errorf("could not get-or-create case: exhausted retries under sustained conflict")

// ErrDeadlockExhausted is returned when all maxDeadlockRetries attempts
// to run the GetOrCreate transaction collide with a MySQL deadlock
// (VOIP-1232). Distinct from ErrGetOrCreateExhausted (which covers the
// narrower uq_case_open_peer insert-conflict retry) so callers
// (subscribehandler.processEvent) can tag deadlock-exhaustion separately
// for interim triage -- see VOIP-1233 for the ack-after-process/DLQ
// follow-up that would give this failure a genuine recovery path instead
// of falling through the current silent-drop pipeline.
var ErrDeadlockExhausted = fmt.Errorf("could not get-or-create case: exhausted retries under sustained deadlock")

// GetOrCreate implements design doc §4's get-or-create algorithm exactly.
//
// VOIP-1232: wraps the whole transaction lifecycle in two additional
// layers versus the original design: (1) a per-peer-tuple in-process
// serialization lock (acquired once here, held across ALL
// maxDeadlockRetries attempts below, released immediately after a
// successful tx.Commit() and BEFORE linkSiblingConversation's network
// RPCs), and (2) an outer bounded retry loop that discards the tx and
// restarts from a fresh BeginTx whenever the driver reports a deadlock.
func (h *caseHandler) GetOrCreate(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType string,
	caseIDHint *uuid.UUID,
) (*kase.Case, error) {
	if peer.Type == "" || peer.Target == "" {
		return nil, cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"CASE_PEER_REQUIRED",
			"peer.type and peer.target are required and cannot be empty.",
		)
	}

	lockKey := peerLockKey(customerID, peer.Type, peer.Target, referenceType)
	release, err := h.acquirePeerLock(ctx, lockKey)
	if err != nil {
		promPeerLockTimeoutTotal.Inc()
		return nil, fmt.Errorf("could not acquire peer lock. GetOrCreate. err: %w", err)
	}
	// Released as soon as a successful attempt commits (see the
	// released-early path below); this defer is the fallback for every
	// early-return error path so the lock is never leaked.
	locked := true
	defer func() {
		if locked {
			release()
		}
	}()

	var lastErr error
	for attempt := 0; attempt < maxDeadlockRetries; attempt++ {
		res, isNewCase, err := h.getOrCreateAttempt(ctx, customerID, self, peer, referenceType, caseIDHint)
		if err == nil {
			// Release the peer lock immediately after a successful commit,
			// strictly BEFORE linkSiblingConversation's cross-service RPCs
			// (design §4.4's own comment already establishes this
			// principle for the DB-level FOR UPDATE locks; the in-process
			// peer lock follows the identical rule -- see
			// getOrCreateAttempt's call site below).
			release()
			locked = false

			if isNewCase && referenceType != "conversation_message" && self.Type != "" {
				h.linkSiblingConversation(ctx, customerID, self, peer, res.ID)
			}

			return res, nil
		}

		if err == dbhandler.ErrDeadlock {
			promDeadlockRetryTotal.Inc()
			lastErr = err
			continue
		}

		return nil, err
	}

	promDeadlockExhaustedTotal.Inc()
	return nil, fmt.Errorf("%w: %v", ErrDeadlockExhausted, lastErr)
}

// getOrCreateAttempt runs ONE full attempt of the GetOrCreate transaction
// lifecycle (BeginTx through Commit), re-capturing `now` fresh for this
// attempt -- so TMCreate/TMUpdate/OpenedAt reflect the wall-clock time of
// the attempt that actually succeeds, not a stale timestamp captured
// before an earlier attempt's deadlock + retry backoff.
func (h *caseHandler) getOrCreateAttempt(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType string,
	caseIDHint *uuid.UUID,
) (*kase.Case, bool, error) {
	now := h.utilHandler.TimeNow()

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("could not begin transaction. GetOrCreate. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, isNewCase, err := h.getOrCreateInTx(ctx, tx, customerID, self, peer, referenceType, caseIDHint, now)
	if err != nil {
		return nil, false, err
	}

	if err := h.db.CaseUpdateTMUpdateTx(ctx, tx, res.ID, now); err != nil {
		if err == dbhandler.ErrDeadlock {
			return nil, false, dbhandler.ErrDeadlock
		}
		return nil, false, fmt.Errorf("could not bump tm_update. GetOrCreate. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("could not commit transaction. GetOrCreate. err: %v", err)
	}
	committed = true

	res.TMUpdate = now
	return res, isNewCase, nil
}

// linkSiblingConversation implements design §4.4's proactive-link write:
// look up (get-only, never create) the sibling message Conversation for
// (self, peer); if found, stamp Metadata.ContactCaseID on it via
// ConversationUpdateMetadata. Both RPCs are best-effort -- see GetOrCreate's
// call site comment for the failure-handling rationale.
func (h *caseHandler) linkSiblingConversation(
	ctx context.Context,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	caseID uuid.UUID,
) {
	conv, err := h.reqHandler.ConversationV1ConversationGetBySelfAndPeer(ctx, self, peer)
	if err != nil || conv == nil {
		// Not found (or lookup failed) -- no further RPC, nothing created.
		// This is the expected, common outcome (no prior message history
		// with this peer), not a defect; do not log as an error.
		return
	}

	id := caseID
	metadata := cvconversation.Metadata{ContactCaseID: &id}
	if _, err := h.reqHandler.ConversationV1ConversationUpdateMetadata(ctx, conv.ID, metadata); err != nil {
		logrus.WithError(err).Warnf("could not update conversation metadata for proactive case link. conversation_id: %s case_id: %s", conv.ID, caseID)
	}
}

// getOrCreateInTx runs design §4 steps 1 (case resolution) inside tx.
// Returns the resolved Case and whether it was freshly inserted this call
// (vs. reused/hint-matched).
func (h *caseHandler) getOrCreateInTx(
	ctx context.Context,
	tx *sql.Tx,
	customerID uuid.UUID,
	self, peer commonaddress.Address,
	referenceType string,
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
			if err == dbhandler.ErrDeadlock {
				return nil, false, dbhandler.ErrDeadlock
			}
			return nil, false, fmt.Errorf("could not validate case_id hint. GetOrCreate. err: %v", err)
		}
		if hinted != nil {
			return hinted, false, nil
		}
		// fall through
	}

	// Step 1b: peer/reference_type resolution.
	found, err := h.db.CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, referenceType)
	if err != nil {
		if err == dbhandler.ErrDeadlock {
			return nil, false, dbhandler.ErrDeadlock
		}
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
		if _, err := h.db.CaseUpdateStatusClosedTx(ctx, tx, customerID, found.ID, kase.ClosedReasonTimeout, kase.ClosedByTypeSystem, nil, now); err != nil {
			if err == dbhandler.ErrDeadlock {
				return nil, false, dbhandler.ErrDeadlock
			}
			return nil, false, fmt.Errorf("could not close timed-out case. GetOrCreate. err: %v", err)
		}
		return h.insertWithRetry(ctx, tx, customerID, self, peer, referenceType, &found.ID, now)
	}

	// Step 1c: no open case at all -- fresh insert, chained to the last
	// closed case for this peer (if any) via previous_case_id.
	lastClosed, err := h.db.CaseGetLastClosedByPeerTx(ctx, tx, customerID, peer.Type, peer.Target, referenceType)
	if err != nil {
		if err == dbhandler.ErrDeadlock {
			return nil, false, dbhandler.ErrDeadlock
		}
		return nil, false, fmt.Errorf("could not look up last closed case. GetOrCreate. err: %v", err)
	}
	var previousCaseID *uuid.UUID
	if lastClosed != nil {
		previousCaseID = &lastClosed.ID
	}
	return h.insertWithRetry(ctx, tx, customerID, self, peer, referenceType, previousCaseID, now)
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
	self, peer commonaddress.Address,
	referenceType string,
	previousCaseID *uuid.UUID,
	now *time.Time,
) (*kase.Case, bool, error) {
	for attempt := 0; attempt < maxInsertRetries; attempt++ {
		newCase := &kase.Case{
			ID:             h.utilHandler.UUIDCreate(),
			CustomerID:     customerID,
			Peer:           peer,
			Local:          self,
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
		if err == dbhandler.ErrDeadlock {
			return nil, false, dbhandler.ErrDeadlock
		}
		if err != dbhandler.ErrDuplicate {
			return nil, false, fmt.Errorf("could not insert case. GetOrCreate. err: %v", err)
		}

		// Conflict: another transaction won. Re-select the winner, locked.
		winner, selErr := h.db.CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, referenceType)
		if selErr != nil {
			if selErr == dbhandler.ErrDeadlock {
				return nil, false, dbhandler.ErrDeadlock
			}
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
