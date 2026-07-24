package servicehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// ServiceAgentInteractionList sends a request to contact-manager
// to list interactions matching the given filter, for the service agent's customer.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero,
// UNLESS all three are zero, in which case the full customer history is listed,
// scoped to the last `since` (default/max enforced by the server layer, §3.5 of the design doc).
// Returns peer_events (peerevent.PeerEvent) unmodified — no reshaping (design doc §8.1/§9).
func (h *serviceHandler) ServiceAgentInteractionList(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
	since time.Time,
) ([]*tmpeerevent.PeerEvent, string, error) {
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
		"since":       since,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll) {
		log.Info("The agent has no permission.")
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1InteractionList(ctx, a.CustomerID, size, token, peerType, peerTarget, contactID, addressID, since)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}
