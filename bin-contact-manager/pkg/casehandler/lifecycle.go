package casehandler

import (
	"context"
	"fmt"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// ErrCaseNotClosed is returned by Continue when the source case is not
// status='closed' (design §5.3 requires a closed source).
var ErrCaseNotClosed = fmt.Errorf("case is not closed; /continue requires a closed source case")

// ErrCaseContinueForbidden is returned by Continue when the caller is
// neither the source case's owning agent nor an admin/manager (design
// §5.3's authorization rule). callerIsAdmin is decided by the API layer's
// permission gate (e.g. PermissionProjectSuperAdmin), not by casehandler
// itself -- casehandler only knows about Case ownership, not the
// platform's broader agent/permission model.
var ErrCaseContinueForbidden = fmt.Errorf("caller is neither the case's owning agent nor an admin/manager")

// CloseResult is the response shape for Close (design §5.1). It always
// reflects the ACTUALLY persisted closed_reason/closed_by -- never the
// caller's own request intent -- distinguishing a genuine first-close
// from a race-lost double-close via AlreadyClosed.
type CloseResult struct {
	Case          *kase.Case
	ClosedReason  string
	ClosedByType  string
	ClosedByID    *uuid.UUID
	AlreadyClosed bool
}

// Close implements design §5.1: an idempotent, race-tolerant close guarded
// by WHERE status='open'. Distinguishes "0 rows because already closed by
// someone/something else" (returns the truthful persisted state) from
// "0 rows because the id doesn't exist" (returns dbhandler.ErrNotFound).
func (h *caseHandler) Close(ctx context.Context, customerID, id uuid.UUID, closedByType commonidentity.OwnerType, closedByID uuid.UUID) (*CloseResult, error) {
	now := h.utilHandler.TimeNow()

	byType := string(closedByType)
	byID := closedByID
	ok, err := h.db.CaseUpdateStatusClosed(ctx, id, kase.ClosedReasonAgentClosed, byType, &byID, now)
	if err != nil {
		return nil, fmt.Errorf("could not close case. Close. err: %v", err)
	}

	c, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.CustomerID != customerID {
		// Tenant isolation: a case belonging to a different customer must
		// behave identically to a non-existent one, never leak existence.
		return nil, dbhandler.ErrNotFound
	}

	if ok {
		return &CloseResult{
			Case:          c,
			ClosedReason:  c.ClosedReason,
			ClosedByType:  c.ClosedByType,
			ClosedByID:    c.ClosedByID,
			AlreadyClosed: false,
		}, nil
	}

	// 0 rows affected: either the case was already closed by someone/
	// something else (truthful state already reflected in the re-read c
	// above), or the update's WHERE id=? never matched at all because the
	// row doesn't exist -- but CaseGetByID above would have already
	// returned ErrNotFound in that case, so reaching here means the row
	// exists and was simply not 'open'.
	return &CloseResult{
		Case:          c,
		ClosedReason:  c.ClosedReason,
		ClosedByType:  c.ClosedByType,
		ClosedByID:    c.ClosedByID,
		AlreadyClosed: true,
	}, nil
}

// Continue implements design §5.3: agent-initiated manual continuation
// for accidental-close recovery. Requires the source case to be
// status='closed'; requires the caller to be the source case's owning
// agent (callerIsAdmin=false path) or an admin/manager
// (callerIsAdmin=true, decided upstream by the API layer's permission
// gate). Creates a brand-new open Case with previous_case_id = id, using
// the same (peer_type, peer_target, reference_type, contact_id) as the
// source -- the source case itself is never modified. Reuses the exact
// same insertWithRetry primitive as §4's get-or-create insert branches
// (not a separate implementation), since /continue is subject to the
// identical uq_case_open_peer race (§5.3).
func (h *caseHandler) Continue(ctx context.Context, customerID, id uuid.UUID, callerType commonidentity.OwnerType, callerID uuid.UUID, callerIsAdmin bool) (*kase.Case, error) {
	source, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if source.CustomerID != customerID {
		return nil, dbhandler.ErrNotFound
	}
	if source.Status != kase.StatusClosed {
		return nil, ErrCaseNotClosed
	}

	if !callerIsAdmin {
		isOwner := source.OwnerType == callerType && source.OwnerID == callerID
		if !isOwner {
			return nil, ErrCaseContinueForbidden
		}
	}

	now := h.utilHandler.TimeNow()

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction. Continue. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	res, _, err := h.insertWithRetry(ctx, tx, customerID, source.PeerType, source.PeerTarget, source.ReferenceType, &source.ID, now)
	if err != nil {
		return nil, err
	}
	res.ContactID = source.ContactID

	if source.ContactID != nil {
		if err := h.db.CaseUpdateContactIDTx(ctx, tx, res.ID, *source.ContactID); err != nil {
			return nil, fmt.Errorf("could not carry over contact_id. Continue. err: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction. Continue. err: %v", err)
	}
	committed = true

	return res, nil
}
