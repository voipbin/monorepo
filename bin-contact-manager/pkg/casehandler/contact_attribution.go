package casehandler

import (
	"context"
	"database/sql"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// deriveCaseContactID implements design §3.4's single derivation
// function -- the ONLY place Case.contact_id is computed. Returns the
// contact_id of the active (tm_delete IS NULL), case-level
// (interaction_id IS NULL), positive Resolution for this case, or nil
// if none exists. customerID scopes the underlying dbhandler query;
// callers must supply the case's own (already-known/validated)
// customer_id -- this function does not itself look up the Case row,
// so it works standalone within an in-progress transaction where the
// Case may not be independently re-queryable yet (e.g. immediately
// after an insert in the same tx).
func (h *caseHandler) deriveCaseContactID(ctx context.Context, customerID, caseID uuid.UUID) (*uuid.UUID, error) {
	resolutions, err := h.db.ResolutionListByCase(ctx, customerID, caseID)
	if err != nil {
		return nil, fmt.Errorf("could not list resolutions. deriveCaseContactID. err: %v", err)
	}
	return firstCaseLevelPositiveContactID(resolutions), nil
}

// deriveCaseContactIDTx is deriveCaseContactID scoped to a caller-managed
// transaction, used by the write paths below so the derivation read and
// the resulting Case.contact_id write happen atomically with the
// triggering Resolution write (design §3.3's single-transaction
// requirement).
func (h *caseHandler) deriveCaseContactIDTx(ctx context.Context, tx *sql.Tx, customerID, caseID uuid.UUID) (*uuid.UUID, error) {
	resolutions, err := h.db.ResolutionListByCaseTx(ctx, tx, customerID, caseID)
	if err != nil {
		return nil, fmt.Errorf("could not list resolutions. deriveCaseContactIDTx. err: %v", err)
	}
	return firstCaseLevelPositiveContactID(resolutions), nil
}

func firstCaseLevelPositiveContactID(resolutions []*resolution.Resolution) *uuid.UUID {
	for _, r := range resolutions {
		if r.ResolutionType == resolution.ResolutionTypePositive && r.InteractionID == nil && r.TMDelete == nil {
			contactID := r.ContactID
			return &contactID
		}
	}
	return nil
}

// applyDerivedContactID writes the result of deriveCaseContactIDTx to
// Case.contact_id: a non-nil derivation writes that contact_id; a nil
// derivation reverts Case.contact_id to NULL (e.g. the sole active
// case-level positive Resolution was just soft-deleted).
func (h *caseHandler) applyDerivedContactID(ctx context.Context, tx *sql.Tx, customerID, caseID uuid.UUID, derived *uuid.UUID) error {
	if derived == nil {
		return h.db.CaseClearContactIDTx(ctx, tx, customerID, caseID)
	}
	return h.db.CaseUpdateContactIDTx(ctx, tx, customerID, caseID, *derived)
}

// ResolutionCreateCaseLevel implements design §3.4's write-path call
// site 1 (create direction): creates a case-level positive or negative
// Resolution and, in the SAME transaction, derives and writes
// Case.contact_id from the result. Verifies the Case belongs to
// customerID before writing -- without this check an attacker who knew
// another tenant's case_id could overwrite that case's contact_id
// (round-2 review defect). Also verifies the target Contact belongs to
// customerID (VOIP-1252 design review round 1 finding) -- without this
// check an agent of one tenant could attach their own Case to an
// arbitrary Contact belonging to a different tenant, mirroring the
// existing check in contacthandler.ResolutionCreate (the
// interaction-level counterpart).
func (h *caseHandler) ResolutionCreateCaseLevel(ctx context.Context, customerID, caseID, contactID uuid.UUID, resolutionType, resolvedByType string, resolvedByID uuid.UUID) (*resolution.Resolution, error) {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return nil, err
	}

	ct, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, fmt.Errorf("could not get contact. ResolutionCreateCaseLevel. err: %v", err)
	}
	if ct.CustomerID != customerID {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	now := h.utilHandler.TimeNow()

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction. ResolutionCreateCaseLevel. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	r := &resolution.Resolution{
		ID:             h.utilHandler.UUIDCreate(),
		CustomerID:     customerID,
		ContactID:      contactID,
		CaseID:         &caseID,
		ResolutionType: resolutionType,
		ResolvedByType: resolvedByType,
		ResolvedByID:   resolvedByID,
		TMCreate:       now,
	}
	if err := h.db.ResolutionCreateTx(ctx, tx, r); err != nil {
		return nil, fmt.Errorf("could not create resolution. ResolutionCreateCaseLevel. err: %v", err)
	}

	derived, err := h.deriveCaseContactIDTx(ctx, tx, customerID, caseID)
	if err != nil {
		return nil, err
	}
	if err := h.applyDerivedContactID(ctx, tx, customerID, caseID, derived); err != nil {
		return nil, fmt.Errorf("could not update case contact_id. ResolutionCreateCaseLevel. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction. ResolutionCreateCaseLevel. err: %v", err)
	}
	committed = true

	return r, nil
}

// ResolutionDeleteCaseLevel implements design §3.4's write-path call
// site 1 (soft-delete direction): soft-deletes a case-level Resolution
// and, in the SAME transaction, re-derives and writes Case.contact_id.
// Verifies the Case belongs to customerID before writing (see
// ResolutionCreateCaseLevel's comment on why this check is required).
func (h *caseHandler) ResolutionDeleteCaseLevel(ctx context.Context, customerID, caseID, id uuid.UUID) error {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return err
	}

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("could not begin transaction. ResolutionDeleteCaseLevel. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := h.db.ResolutionDeleteByCaseTx(ctx, tx, customerID, caseID, id); err != nil {
		return err
	}

	derived, err := h.deriveCaseContactIDTx(ctx, tx, customerID, caseID)
	if err != nil {
		return err
	}
	if err := h.applyDerivedContactID(ctx, tx, customerID, caseID, derived); err != nil {
		return fmt.Errorf("could not update case contact_id. ResolutionDeleteCaseLevel. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. ResolutionDeleteCaseLevel. err: %v", err)
	}
	committed = true

	return nil
}

// CaseListUnresolved implements design §6's agent-facing unresolved
// queue: WHERE customer_id=? AND status='open' AND contact_id IS NULL,
// backed by idx_case_unresolved. Thin delegation to the dbhandler
// primitive already implemented in Task 3.2.
func (h *caseHandler) CaseListUnresolved(ctx context.Context, customerID uuid.UUID) ([]*kase.Case, error) {
	return h.db.CaseListUnresolved(ctx, customerID)
}

// CaseListAll returns every Case (all tenants), for case-control's
// `--all` reconcile-contact sweep. CLI-only usage -- never exposed via
// a customer-facing RPC/route.
func (h *caseHandler) CaseListAll(ctx context.Context) ([]*kase.Case, error) {
	return h.db.CaseListAll(ctx)
}

// ReconcileContact implements design §3.4's recovery path: the
// case-control CLI command `case reconcile-contact <case_id | --all>`
// re-runs deriveCaseContactID and overwrites Case.contact_id. Idempotent
// -- used only if drift is discovered (e.g. a bulk import wrote
// Resolution rows without going through the handler). Runs inside a
// transaction for the same reason ResolutionCreateCaseLevel/
// ResolutionDeleteCaseLevel do: the derivation read and the resulting
// Case.contact_id write must be atomic.
func (h *caseHandler) ReconcileContact(ctx context.Context, caseID uuid.UUID) error {
	c, err := h.db.CaseGetByID(ctx, caseID)
	if err != nil {
		return err
	}

	tx, err := h.db.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("could not begin transaction. ReconcileContact. err: %v", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	derived, err := h.deriveCaseContactIDTx(ctx, tx, c.CustomerID, caseID)
	if err != nil {
		return err
	}
	if err := h.applyDerivedContactID(ctx, tx, c.CustomerID, caseID, derived); err != nil {
		return fmt.Errorf("could not update case contact_id. ReconcileContact. err: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction. ReconcileContact. err: %v", err)
	}
	committed = true

	return nil
}
