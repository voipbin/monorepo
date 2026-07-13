package casehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// verifyCaseOwnershipAndGet confirms the given Case belongs to
// customerID, returning dbhandler.ErrNotFound if it doesn't exist or
// belongs to a different tenant -- never leaking existence of another
// tenant's case. This is the shared tenant-isolation choke point
// (design §4 step 1's customer_id predicate, generalized to every
// case-scoped mutation, not just get-or-create): round-2 review found
// CaseTagAdd/Remove/List, CaseNoteCreate, and UpdateContact (VOIP-1253)
// all accepted a customerID parameter but never actually used it to
// gate the mutation -- an attacker who knew or guessed another
// tenant's case_id could tag, note, or attribute it. Call this FIRST,
// before any mutating dbhandler call, in every case-scoped handler
// method.
//
// Returns the fetched *kase.Case (VOIP-1254) so callers that need the
// Case's current fields (e.g. TagIDs for a read-modify-write) can reuse
// this same row instead of a second CaseGetByID round trip. Callers
// that only need the ownership check may discard the returned Case.
func verifyCaseOwnershipAndGet(ctx context.Context, db dbhandler.DBHandler, customerID, caseID uuid.UUID) (*kase.Case, error) {
	c, err := db.CaseGetByID(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if c.CustomerID != customerID {
		return nil, dbhandler.ErrNotFound
	}
	return c, nil
}

// ErrTagNotFound is returned by CaseTagAdd when the given tag_id does
// not exist in bin-tag-manager.
var ErrTagNotFound = fmt.Errorf("tag not found")

// containsUUID reports whether target is present in slice. New helper
// introduced by design VOIP-1254 (not a pre-existing utility).
func containsUUID(slice []uuid.UUID, target uuid.UUID) bool {
	for _, id := range slice {
		if id == target {
			return true
		}
	}
	return false
}

// CaseTagAdd implements design VOIP-1254: assigns a tag to a Case by
// appending to Case.TagIDs (a plain JSON column, mirroring
// bin-queue-manager's Queue.TagIDs storage exactly -- no junction
// table, no reverse lookup). Validates tag_id existence via
// bin-tag-manager's existing TagV1TagGet before writing -- no other
// tag-manager interaction needed, and bin-tag-manager itself is
// unchanged (Cases reference the same Tag rows Contacts and Queues
// already do). Also verifies the Case belongs to customerID before
// either check.
//
// Idempotent: adding a tag_id already present in TagIDs is a no-op --
// no dbhandler write, no case_tag_added event.
//
// Not atomic against a concurrent second CaseTagAdd/Remove on the same
// case (read-modify-write, no SELECT...FOR UPDATE) -- an accepted,
// narrow lost-update window for this low-frequency, single-agent UI
// action, per design VOIP-1254's Concurrency note.
func (h *caseHandler) CaseTagAdd(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID)
	if err != nil {
		return err
	}

	if _, err := h.reqHandler.TagV1TagGet(ctx, tagID); err != nil {
		return ErrTagNotFound
	}

	if containsUUID(c.TagIDs, tagID) {
		return nil // idempotent no-op, already tagged -- no write, no event
	}

	newTagIDs := append(append([]uuid.UUID{}, c.TagIDs...), tagID)
	if err := h.db.CaseUpdateTagIDs(ctx, customerID, caseID, newTagIDs); err != nil {
		return fmt.Errorf("could not update case tag_ids. CaseTagAdd. err: %v", err)
	}

	h.notifyHandler.PublishEvent(ctx, "case_tag_added", map[string]uuid.UUID{
		"case_id": caseID,
		"tag_id":  tagID,
	})

	return nil
}

// CaseTagRemove implements design VOIP-1254: unassigns a tag from a
// Case by filtering it out of Case.TagIDs. Verifies case ownership
// first (see CaseTagAdd's comment on why this check is required).
//
// Idempotent, explicitly symmetric with CaseTagAdd's no-op: removing an
// absent tag_id is a no-op, not an error -- no dbhandler write (nothing
// to persist), no case_tag_removed event (firing a "removed" event for
// a tag that was never present would be a semantically false audit
// record).
func (h *caseHandler) CaseTagRemove(ctx context.Context, customerID, caseID, tagID uuid.UUID) error {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID)
	if err != nil {
		return err
	}

	if !containsUUID(c.TagIDs, tagID) {
		return nil // idempotent no-op, tag was never present -- no write, no event
	}

	newTagIDs := make([]uuid.UUID, 0, len(c.TagIDs)-1)
	for _, id := range c.TagIDs {
		if id != tagID {
			newTagIDs = append(newTagIDs, id)
		}
	}

	if err := h.db.CaseUpdateTagIDs(ctx, customerID, caseID, newTagIDs); err != nil {
		return fmt.Errorf("could not update case tag_ids. CaseTagRemove. err: %v", err)
	}

	h.notifyHandler.PublishEvent(ctx, "case_tag_removed", map[string]uuid.UUID{
		"case_id": caseID,
		"tag_id":  tagID,
	})

	return nil
}

// CaseTagList returns all tag IDs assigned to a Case. Verifies case
// ownership first (see CaseTagAdd's comment on why this check is
// required). No separate dbhandler call -- TagIDs is already part of
// the Case row returned by the ownership check (design VOIP-1254).
func (h *caseHandler) CaseTagList(ctx context.Context, customerID, caseID uuid.UUID) ([]uuid.UUID, error) {
	c, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID)
	if err != nil {
		return nil, err
	}
	return c.TagIDs, nil
}
