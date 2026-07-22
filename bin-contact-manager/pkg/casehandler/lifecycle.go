package casehandler

import (
	"context"
	"fmt"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// ErrCaseNotClosed is returned by Continue when the source case is not
// status='closed' (design §5.3 requires a closed source). Typed as a
// *cerrors.VoipbinError (InvalidArgument -> HTTP 400) so listenhandler's
// errorResponse() maps it correctly instead of falling through to a
// generic 500 -- matching the OpenAPI spec's declared 400 response for
// POST /v1.0/contact_cases/{id}/continue.
var ErrCaseNotClosed = cerrors.InvalidArgument(
	commonoutline.ServiceNameContactManager,
	"CASE_NOT_CLOSED",
	"case is not closed; /continue requires a closed source case",
)

// ErrCaseContinueForbidden is returned by Continue when the caller is
// neither the source case's owning agent nor an admin/manager (design
// §5.3's authorization rule). callerIsAdmin is decided by the API layer's
// permission gate (e.g. PermissionProjectSuperAdmin), not by casehandler
// itself -- casehandler only knows about Case ownership, not the
// platform's broader agent/permission model. Typed as a
// *cerrors.VoipbinError (PermissionDenied -> HTTP 403) so listenhandler's
// errorResponse() maps it correctly instead of falling through to a
// generic 500 -- matching the OpenAPI spec's declared 403 response for
// POST /v1.0/contact_cases/{id}/continue.
var ErrCaseContinueForbidden = cerrors.PermissionDenied(
	commonoutline.ServiceNameContactManager,
	"CASE_CONTINUE_FORBIDDEN",
	"caller is neither the case's owning agent nor an admin/manager",
)

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
// by WHERE customer_id=? AND status='open' (the tenant predicate lives in
// the mutating UPDATE statement itself, not in a separate check run after
// the mutation -- a cross-tenant caller's UPDATE can never match a row at
// all). Distinguishes "0 rows because already closed by someone/something
// else" (returns the truthful persisted state) from "0 rows because the id
// doesn't exist or belongs to a different tenant" (returns
// dbhandler.ErrNotFound).
func (h *caseHandler) Close(ctx context.Context, customerID, id uuid.UUID, closedByType commonidentity.OwnerType, closedByID uuid.UUID) (*CloseResult, error) {
	now := h.utilHandler.TimeNow()

	byType := string(closedByType)
	byID := closedByID
	ok, err := h.db.CaseUpdateStatusClosed(ctx, customerID, id, kase.ClosedReasonAgentClosed, byType, &byID, now)
	if err != nil {
		return nil, fmt.Errorf("could not close case. Close. err: %v", err)
	}

	if ok {
		c, err := h.db.CaseGetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &CloseResult{
			Case:          c,
			ClosedReason:  c.ClosedReason,
			ClosedByType:  c.ClosedByType,
			ClosedByID:    c.ClosedByID,
			AlreadyClosed: false,
		}, nil
	}

	// 0 rows affected: the UPDATE's WHERE id=? AND customer_id=? AND
	// status='open' matched nothing. This is ambiguous between three
	// cases -- id doesn't exist at all, id exists but belongs to a
	// different tenant, or id exists in this tenant but was already
	// closed -- and re-reading via the tenant-scoped path below resolves
	// the ambiguity without ever having mutated a row we didn't own.
	c, err := h.db.CaseGetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.CustomerID != customerID {
		// Tenant isolation: a case belonging to a different customer must
		// behave identically to a non-existent one, never leak existence.
		return nil, dbhandler.ErrNotFound
	}

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

	res, _, err := h.insertWithRetry(ctx, tx, customerID, commonaddress.Address{}, source.Peer, source.ReferenceType, &source.ID, now)
	if err != nil {
		return nil, err
	}
	res.ContactID = source.ContactID

	if source.ContactID != nil {
		if err := h.db.CaseUpdateContactIDTx(ctx, tx, customerID, res.ID, *source.ContactID); err != nil {
			return nil, fmt.Errorf("could not carry over contact_id. Continue. err: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction. Continue. err: %v", err)
	}
	committed = true

	return res, nil
}
