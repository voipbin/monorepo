package casehandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/casenote"
)

// CaseNoteCreate implements design §3.5: creates an internal, agent-
// facing CaseNote and publishes case_note_created via the plain
// notifyHandler.PublishEvent() primitive -- NEVER PublishWebhookEvent().
// CaseNote must never reach a customer-facing webhook. Verifies the
// Case belongs to customerID before inserting -- without this check an
// attacker who knew another tenant's case_id could attach notes to it
// (round-2 review defect).
func (h *caseHandler) CaseNoteCreate(ctx context.Context, customerID, caseID uuid.UUID, authorType string, authorID *uuid.UUID, text string) (*casenote.CaseNote, error) {
	if err := verifyCaseOwnership(ctx, h.db, customerID, caseID); err != nil {
		return nil, err
	}

	now := h.utilHandler.TimeNow()

	n := &casenote.CaseNote{
		ID:         h.utilHandler.UUIDCreate(),
		CustomerID: customerID,
		CaseID:     caseID,
		AuthorType: authorType,
		AuthorID:   authorID,
		Text:       text,
		TMCreate:   now,
	}

	if err := h.db.CaseNoteCreate(ctx, n); err != nil {
		return nil, fmt.Errorf("could not create case note. CaseNoteCreate. err: %v", err)
	}

	h.notifyHandler.PublishEvent(ctx, "case_note_created", n)

	return n, nil
}

// CaseNoteDelete implements design §3.5: soft-deletes a CaseNote and
// publishes case_note_deleted via the plain PublishEvent() primitive --
// NEVER PublishWebhookEvent(), for the same isolation reason as create.
func (h *caseHandler) CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error {
	if err := h.db.CaseNoteDelete(ctx, customerID, caseID, id); err != nil {
		return err
	}

	h.notifyHandler.PublishEvent(ctx, "case_note_deleted", map[string]uuid.UUID{
		"id":          id,
		"case_id":     caseID,
		"customer_id": customerID,
	})

	return nil
}

// CaseNoteListByCase is a thin delegation to the dbhandler primitive.
func (h *caseHandler) CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error) {
	return h.db.CaseNoteListByCase(ctx, customerID, caseID)
}
