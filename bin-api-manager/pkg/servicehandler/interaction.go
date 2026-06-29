package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"
)

// InteractionList sends a request to contact-manager
// to list interactions matching the given filter.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero.
func (h *serviceHandler) InteractionList(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
) (*cminteraction.InteractionListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "InteractionList",
		"customer_id": a.CustomerID,
		"peer_type":   peerType,
		"peer_target": peerTarget,
		"contact_id":  contactID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1InteractionList(ctx, a.CustomerID, size, token, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		return nil, err
	}

	return res, nil
}

// InteractionListUnresolved sends a request to contact-manager
// to list interactions with no active resolution within the given lookback window.
func (h *serviceHandler) InteractionListUnresolved(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	sinceDays int,
) (*cminteraction.InteractionListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "InteractionListUnresolved",
		"customer_id": a.CustomerID,
		"since_days":  sinceDays,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1InteractionListUnresolved(ctx, a.CustomerID, size, token, sinceDays)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		return nil, err
	}

	return res, nil
}

// InteractionGet sends a request to contact-manager
// to get a single interaction by ID.
func (h *serviceHandler) InteractionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cminteraction.Interaction, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "InteractionGet",
		"customer_id":    a.CustomerID,
		"interaction_id": id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1InteractionGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the interaction. err: %v", err)
		return nil, err
	}

	// Ownership check: interaction must belong to this customer.
	if res.CustomerID != a.CustomerID {
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// ResolutionCreate sends a request to contact-manager
// to create a resolution for an interaction.
func (h *serviceHandler) ResolutionCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	interactionID uuid.UUID,
	contactID uuid.UUID,
	resolutionType string,
	resolvedByType string,
	resolvedByID uuid.UUID,
) (*cmresolution.Resolution, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "ResolutionCreate",
		"customer_id":     a.CustomerID,
		"interaction_id":  interactionID,
		"contact_id":      contactID,
		"resolution_type": resolutionType,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ResolutionCreate(ctx, interactionID, a.CustomerID, contactID, resolutionType, resolvedByType, resolvedByID)
	if err != nil {
		log.Errorf("Could not create resolution. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ResolutionDelete sends a request to contact-manager
// to soft-delete a resolution.
func (h *serviceHandler) ResolutionDelete(ctx context.Context, a *auth.AuthIdentity, interactionID, resolutionID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "ResolutionDelete",
		"customer_id":    a.CustomerID,
		"interaction_id": interactionID,
		"resolution_id":  resolutionID,
	})

	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return serviceerrors.ErrPermissionDenied
	}

	if err := h.reqHandler.ContactV1ResolutionDelete(ctx, a.CustomerID, interactionID, resolutionID); err != nil {
		log.Errorf("Could not delete resolution. err: %v", err)
		return err
	}

	return nil
}
