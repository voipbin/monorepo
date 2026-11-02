package casehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// CaseNoteCreate implements design §3.5: creates an internal, agent-
// facing CaseNote and publishes case_note_created via the plain
// notifyHandler.PublishEvent() primitive -- NEVER PublishWebhookEvent().
// CaseNote must never reach a customer-facing webhook. Verifies the
// Case belongs to customerID before inserting -- without this check an
// attacker who knew another tenant's case_id could attach notes to it
// (round-2 review defect).
func (h *caseHandler) CaseNoteCreate(ctx context.Context, customerID, caseID uuid.UUID, authorType string, authorID *uuid.UUID, text string) (*casenote.CaseNote, error) {
	if _, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID); err != nil {
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
		return nil, fmt.Errorf("could not create case note. CaseNoteCreate. err: %w", err)
	}

	h.notifyHandler.PublishEvent(ctx, casenote.EventTypeCaseNoteCreated, n)

	return n, nil
}

// CaseNoteDelete implements design §3.5: soft-deletes a CaseNote and
// publishes case_note_deleted via the plain PublishEvent() primitive --
// NEVER PublishWebhookEvent(), for the same isolation reason as create.
func (h *caseHandler) CaseNoteDelete(ctx context.Context, customerID, caseID, id uuid.UUID) error {
	if err := h.db.CaseNoteDelete(ctx, customerID, caseID, id); err != nil {
		return err
	}

	h.notifyHandler.PublishEvent(ctx, casenote.EventTypeCaseNoteDeleted, &casenote.CaseNoteDeletedEvent{
		ID:         id,
		CaseID:     caseID,
		CustomerID: customerID,
	})

	return nil
}

// CaseNoteListByCase is a thin delegation to the dbhandler primitive.
func (h *caseHandler) CaseNoteListByCase(ctx context.Context, customerID, caseID uuid.UUID) ([]*casenote.CaseNote, error) {
	return h.db.CaseNoteListByCase(ctx, customerID, caseID)
}

// CaseNoteListByContact lists the notes written on every case currently
// attributed to the contact, newest first (VOIP-1673 A1). The contact must
// belong to customerID and not be deleted; otherwise the same NotFound a
// missing contact produces is returned, so another tenant's contact id
// never reveals its existence. Pagination follows CaseList: size+1 rows
// are read to detect a further page and the token is the tm_create of the
// last returned note (empty when there is no further page).
func (h *caseHandler) CaseNoteListByContact(ctx context.Context, customerID, contactID uuid.UUID, size uint64, token string) ([]*casenote.CaseNote, string, error) {
	ct, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, "", cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, "", fmt.Errorf("could not get contact. CaseNoteListByContact. err: %w", err)
	}
	if ct.CustomerID != customerID || ct.TMDelete != nil {
		return nil, "", cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	if size == 0 {
		size = defaultCaseListSize
	}

	items, err := h.db.CaseNoteListByContact(ctx, customerID, contactID, size+1, token)
	if err != nil {
		return nil, "", err
	}

	hasMore := uint64(len(items)) > size
	if hasMore {
		items = items[:size]
	}

	var nextToken string
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		if last.TMCreate != nil {
			nextToken = last.TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	return items, nextToken, nil
}
