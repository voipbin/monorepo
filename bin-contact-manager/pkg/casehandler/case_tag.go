package casehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
)

// ErrTagNotFound is returned by CaseTagAdd when the given tag_id does
// not exist in bin-tag-manager.
var ErrTagNotFound = fmt.Errorf("tag not found")

// CaseTagAdd implements design §7 round-22 correction: assigns a tag to
// a Case. Validates tag_id existence via bin-tag-manager's existing
// TagV1TagGet before creating the case-scoped assignment row -- no
// other tag-manager interaction needed, and bin-tag-manager itself is
// unchanged (Cases reference the same Tag rows Contacts already do).
func (h *caseHandler) CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	if _, err := h.reqHandler.TagV1TagGet(ctx, tagID); err != nil {
		return ErrTagNotFound
	}

	if err := h.db.CaseTagAssignmentCreate(ctx, caseID, tagID); err != nil {
		return fmt.Errorf("could not create case tag assignment. CaseTagAdd. err: %v", err)
	}

	return nil
}

// CaseTagRemove implements design §7 round-22 correction: unassigns a
// tag from a Case.
func (h *caseHandler) CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	if err := h.db.CaseTagAssignmentDelete(ctx, caseID, tagID); err != nil {
		return fmt.Errorf("could not delete case tag assignment. CaseTagRemove. err: %v", err)
	}
	return nil
}

// CaseTagList returns all tag IDs assigned to a Case.
func (h *caseHandler) CaseTagList(ctx context.Context, customerID, caseID uuid.UUID) ([]uuid.UUID, error) {
	return h.db.CaseTagAssignmentListByCaseID(ctx, caseID)
}
