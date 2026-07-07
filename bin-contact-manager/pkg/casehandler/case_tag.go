package casehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// verifyCaseOwnership confirms the given Case belongs to customerID,
// returning dbhandler.ErrNotFound if it doesn't exist or belongs to a
// different tenant -- never leaking existence of another tenant's case.
// This is the shared tenant-isolation choke point (design §4 step 1's
// customer_id predicate, generalized to every case-scoped mutation, not
// just get-or-create): round-2 review found CaseTagAdd/Remove/List,
// CaseNoteCreate, and ResolutionCreateCaseLevel/ResolutionDeleteCaseLevel
// all accepted a customerID parameter but never actually used it to gate
// the mutation -- an attacker who knew or guessed another tenant's
// case_id could tag, note, or resolve it. Call this FIRST, before any
// mutating dbhandler call, in every case-scoped handler method.
func verifyCaseOwnership(ctx context.Context, db dbhandler.DBHandler, customerID, caseID uuid.UUID) error {
	c, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		return err
	}
	if c.CustomerID != customerID {
		return dbhandler.ErrNotFound
	}
	return nil
}

// ErrTagNotFound is returned by CaseTagAdd when the given tag_id does
// not exist in bin-tag-manager.
var ErrTagNotFound = fmt.Errorf("tag not found")

// CaseTagAdd implements design §7 round-22 correction: assigns a tag to
// a Case. Validates tag_id existence via bin-tag-manager's existing
// TagV1TagGet before creating the case-scoped assignment row -- no
// other tag-manager interaction needed, and bin-tag-manager itself is
// unchanged (Cases reference the same Tag rows Contacts already do).
// Also verifies the Case belongs to customerID before either check --
// contact_case_tag_assignments has no customer_id column of its own,
// so this handler-level ownership check is the only tenant guard.
func (h *caseHandler) CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return err
	}

	if _, err := h.reqHandler.TagV1TagGet(ctx, tagID); err != nil {
		return ErrTagNotFound
	}

	if err := h.db.CaseTagAssignmentCreate(ctx, caseID, tagID); err != nil {
		return fmt.Errorf("could not create case tag assignment. CaseTagAdd. err: %v", err)
	}

	return nil
}

// CaseTagRemove implements design §7 round-22 correction: unassigns a
// tag from a Case. Verifies case ownership first (see CaseTagAdd's
// comment on why this check is required).
func (h *caseHandler) CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return err
	}

	if err := h.db.CaseTagAssignmentDelete(ctx, caseID, tagID); err != nil {
		return fmt.Errorf("could not delete case tag assignment. CaseTagRemove. err: %v", err)
	}
	return nil
}

// CaseTagList returns all tag IDs assigned to a Case. Verifies case
// ownership first (see CaseTagAdd's comment on why this check is
// required).
func (h *caseHandler) CaseTagList(ctx context.Context, customerID, caseID uuid.UUID) ([]uuid.UUID, error) {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return nil, err
	}
	return h.db.CaseTagAssignmentListByCaseID(ctx, caseID)
}
