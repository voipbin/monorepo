package casehandler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// maxInsertRetries bounds the ON DUPLICATE KEY retry loop in
// insertWithRetry (design §4.2's round-2 correction: retry, don't assume
// the first re-select is final). Loop exhaustion is an extremely rare
// thundering-herd scenario; see design §4.2's "Loop exhaustion" note.
const maxInsertRetries = 3

// ErrGetOrCreateExhausted is returned when all maxInsertRetries attempts
// to insert-or-reselect an open Case collide (design §4.2's "Loop
// exhaustion" path). The name predates VOIP-1439's removal of
// GetOrCreate; retained to avoid an unrelated API rename in a
// deletion-only change. The only remaining caller is Continue
// (lifecycle.go), which surfaces this as a transient 5xx via
// listenhandler rather than silently dropping the request.
var ErrGetOrCreateExhausted = fmt.Errorf("could not get-or-create case: exhausted retries under sustained conflict")

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
	referenceID string,
) (*kase.Case, bool, error) {
	for attempt := 0; attempt < maxInsertRetries; attempt++ {
		newCase := &kase.Case{
			ID:             h.utilHandler.UUIDCreate(),
			CustomerID:     customerID,
			Peer:           peer,
			Local:          self,
			ReferenceType:  referenceType,
			ReferenceID:    referenceID,
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
			return nil, false, fmt.Errorf("could not insert case. insertWithRetry. err: %v", err)
		}

		// Conflict: another transaction won. Re-select the winner, locked.
		winner, selErr := h.db.CaseGetOpenByPeer(ctx, tx, customerID, peer.Type, peer.Target, referenceType)
		if selErr != nil {
			if selErr == dbhandler.ErrDeadlock {
				return nil, false, dbhandler.ErrDeadlock
			}
			return nil, false, fmt.Errorf("could not re-select after insert conflict. insertWithRetry. err: %v", selErr)
		}
		if winner != nil {
			return winner, false, nil
		}
		// The row we raced against itself transitioned out of 'open'
		// before we re-selected it (rarer second race) -- loop and retry.
	}

	return nil, false, ErrGetOrCreateExhausted
}
