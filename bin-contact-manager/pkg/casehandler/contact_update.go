package casehandler

import (
	"context"
	stderrors "errors"
	"fmt"

	"github.com/gofrs/uuid"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// UpdateContact implements design VOIP-1253: attaches or detaches a
// Case's Contact via a direct Case.contact_id write, replacing
// VOIP-1252's Resolution-based mechanism. contactID == uuid.Nil clears
// the attribution (mirrors bin-conference-manager's PreFlowID/
// PostFlowID PUT convention: empty UUID in the request clears the
// link). Verifies the Case belongs to customerID (mirrors
// verifyCaseOwnership, preserved from VOIP-1252) and, when attaching
// (contactID != uuid.Nil), verifies the target Contact belongs to
// customerID too (preserved from VOIP-1252 round-1 review finding --
// without this check an agent of one tenant could attach their Case to
// another tenant's Contact).
func (h *caseHandler) UpdateContact(ctx context.Context, customerID, caseID, contactID uuid.UUID) (*kase.Case, error) {
	if _, err := verifyCaseOwnershipAndGet(ctx, h.db, customerID, caseID); err != nil {
		return nil, err
	}

	eventType := "case_contact_detached"
	if contactID != uuid.Nil {
		ct, err := h.db.ContactGet(ctx, contactID)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return nil, cerrors.NotFound(
					commonoutline.ServiceNameContactManager,
					"CONTACT_NOT_FOUND",
					"The contact was not found.",
				).Wrap(err)
			}
			return nil, fmt.Errorf("could not get contact. UpdateContact. err: %v", err)
		}
		if ct.CustomerID != customerID {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			)
		}

		if err := h.db.CaseUpdateContactID(ctx, customerID, caseID, contactID); err != nil {
			return nil, fmt.Errorf("could not update case contact_id. UpdateContact. err: %v", err)
		}
		eventType = "case_contact_attributed"
	} else {
		if err := h.db.CaseClearContactID(ctx, customerID, caseID); err != nil {
			return nil, fmt.Errorf("could not clear case contact_id. UpdateContact. err: %v", err)
		}
	}

	c, err := h.db.CaseGetByID(ctx, caseID)
	if err != nil {
		return nil, fmt.Errorf("could not get updated case. UpdateContact. err: %v", err)
	}

	// Audit trail (replaces VOIP-1252's Resolution row): who/when
	// changed this Case's Contact attribution, picked up automatically
	// by bin-timeline-manager (already subscribes to
	// bin-manager.contact-manager.event, zero new wiring needed).
	// Mirrors casenote.go's PublishEvent-only precedent -- this is an
	// internal state-change event, not a customer-facing webhook, so
	// PublishEvent (never PublishWebhookEvent) is correct here too.
	h.notifyHandler.PublishEvent(ctx, eventType, map[string]uuid.UUID{
		"case_id":    caseID,
		"contact_id": contactID, // uuid.Nil on detach -- consumer reads eventType to disambiguate
	})

	return c, nil
}
