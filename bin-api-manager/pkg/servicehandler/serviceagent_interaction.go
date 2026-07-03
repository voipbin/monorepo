package servicehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	cminteraction "monorepo/bin-contact-manager/models/interaction"
	cmresolution "monorepo/bin-contact-manager/models/resolution"
)

// ServiceAgentInteractionList sends a request to contact-manager
// to list interactions matching the given filter, for the service agent's customer.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero
// (same requirement as the top-level Admin/Manager InteractionList).
func (h *serviceHandler) ServiceAgentInteractionList(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
) ([]*cminteraction.Interaction, string, error) {
	if !a.IsAgent() {
		return nil, "", serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentInteractionList",
		"customer_id": a.CustomerID,
		"peer_type":   peerType,
		"peer_target": peerTarget,
		"contact_id":  contactID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1InteractionList(ctx, a.CustomerID, size, token, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}

// ServiceAgentInteractionListUnresolved sends a request to contact-manager
// to list interactions with no active resolution within the given lookback window,
// for the service agent's customer.
func (h *serviceHandler) ServiceAgentInteractionListUnresolved(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	since string,
) ([]*cminteraction.Interaction, string, error) {
	if !a.IsAgent() {
		return nil, "", serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":        "ServiceAgentInteractionListUnresolved",
		"customer_id": a.CustomerID,
		"since":       since,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1InteractionListUnresolved(ctx, a.CustomerID, size, token, since)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}

// ServiceAgentInteractionGet sends a request to contact-manager
// to get a single interaction by ID, for the service agent's customer.
func (h *serviceHandler) ServiceAgentInteractionGet(ctx context.Context, a *auth.AuthIdentity, id uuid.UUID) (*cminteraction.Interaction, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceAgentInteractionGet",
		"customer_id":    a.CustomerID,
		"interaction_id": id,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Fetch first to validate ownership (interaction.CustomerID must match a.CustomerID).
	res, err := h.interactionGet(ctx, a.CustomerID, id)
	if err != nil {
		log.Errorf("Could not get the interaction. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, res.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	return res, nil
}

// ServiceAgentResolutionCreate sends a request to contact-manager
// to create a resolution for an interaction, for the service agent's customer.
func (h *serviceHandler) ServiceAgentResolutionCreate(
	ctx context.Context,
	a *auth.AuthIdentity,
	interactionID uuid.UUID,
	contactID uuid.UUID,
	resolutionType string,
	resolvedByType string,
	resolvedByID uuid.UUID,
) (*cmresolution.Resolution, error) {
	if !a.IsAgent() {
		return nil, serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":            "ServiceAgentResolutionCreate",
		"customer_id":     a.CustomerID,
		"interaction_id":  interactionID,
		"contact_id":      contactID,
		"resolution_type": resolutionType,
	})

	if a.IsDirect() {
		return nil, serviceerrors.ErrDirectAccessNotSupported
	}

	// Fetch interaction first to validate ownership before attaching a resolution.
	ia, err := h.interactionGet(ctx, a.CustomerID, interactionID)
	if err != nil {
		log.Errorf("Could not get the interaction info. err: %v", err)
		return nil, err
	}

	if !h.hasPermission(ctx, a, ia.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, serviceerrors.ErrPermissionDenied
	}

	res, err := h.reqHandler.ContactV1ResolutionCreate(ctx, a.CustomerID, contactID, interactionID, resolutionType, resolvedByType, resolvedByID)
	if err != nil {
		log.Errorf("Could not create resolution. err: %v", err)
		return nil, err
	}

	return res, nil
}

// ServiceAgentResolutionDelete sends a request to contact-manager
// to soft-delete a resolution, for the service agent's customer.
// interactionID is passed through the full chain so the DB enforces
// WHERE customer_id=? AND interaction_id=? AND id=? — preventing cross-interaction deletion.
func (h *serviceHandler) ServiceAgentResolutionDelete(ctx context.Context, a *auth.AuthIdentity, interactionID, resolutionID uuid.UUID) error {
	if !a.IsAgent() {
		return serviceerrors.ErrAuthenticationRequired
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "ServiceAgentResolutionDelete",
		"customer_id":    a.CustomerID,
		"interaction_id": interactionID,
		"resolution_id":  resolutionID,
	})

	if a.IsDirect() {
		return serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return serviceerrors.ErrPermissionDenied
	}

	if err := h.reqHandler.ContactV1ResolutionDelete(ctx, a.CustomerID, interactionID, resolutionID); err != nil {
		log.Errorf("Could not delete resolution. err: %v", err)
		return err
	}

	return nil
}
