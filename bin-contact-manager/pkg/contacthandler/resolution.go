package contacthandler

import (
	"context"
	"fmt"

	stderrors "errors"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// ResolutionCreate creates a new resolution for an interaction.
//
// Validates:
//  1. The interaction must exist and belong to customerID (tenant guard).
//  2. No existing active resolution of the SAME type for (contact_id, interaction_id).
//     A positive + negative coexisting is allowed. Two positives or two negatives are rejected.
//     This is application-level dedup (no DB UNIQUE due to MySQL NULL semantics + soft-delete).
//     Concurrent creates may slip through; v1 accepts this as a UX nuisance, not data corruption.
func (h *contactHandler) ResolutionCreate(
	ctx context.Context,
	customerID, contactID, interactionID uuid.UUID,
	resolutionType, resolvedByType string,
	resolvedByID uuid.UUID,
) (*resolution.Resolution, error) {
	// 1. Verify the interaction exists and belongs to this customer.
	iact, err := h.db.InteractionGet(ctx, interactionID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"INTERACTION_NOT_FOUND",
				"The interaction was not found.",
			).Wrap(err)
		}
		return nil, fmt.Errorf("could not get interaction. ResolutionCreate. err: %v", err)
	}
	if iact.CustomerID != customerID {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"INTERACTION_NOT_FOUND",
			"The interaction was not found.",
		)
	}

	// 2. Verify the contact exists and belongs to this customer.
	ct, err := h.db.ContactGet(ctx, contactID)
	if err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return nil, cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			).Wrap(err)
		}
		return nil, fmt.Errorf("could not get contact. ResolutionCreate. err: %v", err)
	}
	if ct.CustomerID != customerID {
		return nil, cerrors.NotFound(
			commonoutline.ServiceNameContactManager,
			"CONTACT_NOT_FOUND",
			"The contact was not found.",
		)
	}

	// 3. Check for existing active resolution of the same type.
	existing, err := h.db.ResolutionListByInteraction(ctx, customerID, interactionID)
	if err != nil {
		return nil, fmt.Errorf("could not list resolutions. ResolutionCreate. err: %v", err)
	}
	for _, r := range existing {
		if r.ContactID == contactID && r.ResolutionType == resolutionType {
			return nil, cerrors.AlreadyExists(
				commonoutline.ServiceNameContactManager,
				"RESOLUTION_ALREADY_EXISTS",
				fmt.Sprintf("An active %s resolution for this (contact_id, interaction_id) already exists.", resolutionType),
			)
		}
	}

	// 4. Create.
	id := h.utilHandler.UUIDCreate()
	now := h.utilHandler.TimeNow()

	r := &resolution.Resolution{
		ID:             id,
		CustomerID:     customerID,
		ContactID:      contactID,
		InteractionID:  interactionID,
		ResolutionType: resolutionType,
		ResolvedByType: resolvedByType,
		ResolvedByID:   resolvedByID,
		TMCreate:       now,
		TMUpdate:       now,
	}

	if createErr := h.db.ResolutionCreate(ctx, r); createErr != nil {
		return nil, fmt.Errorf("could not create resolution. ResolutionCreate. err: %v", createErr)
	}

	return r, nil
}

// ResolutionDelete soft-deletes a resolution.
// Validates that the resolution exists and belongs to customerID and interactionID.
// Cross-tenant and cross-interaction guard is enforced in dbhandler
// (WHERE customer_id=? AND interaction_id=? AND id=?).
func (h *contactHandler) ResolutionDelete(ctx context.Context, customerID, interactionID, id uuid.UUID) error {
	if err := h.db.ResolutionDelete(ctx, customerID, interactionID, id); err != nil {
		if stderrors.Is(err, dbhandler.ErrNotFound) {
			return cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"RESOLUTION_NOT_FOUND",
				"The resolution was not found or is already deleted.",
			).Wrap(err)
		}
		return fmt.Errorf("could not delete resolution. ResolutionDelete. err: %v", err)
	}

	return nil
}
