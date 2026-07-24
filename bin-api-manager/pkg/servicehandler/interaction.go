package servicehandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"

	amagent "monorepo/bin-agent-manager/models/agent"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// InteractionList sends a request to contact-manager
// to list interactions matching the given filter.
// Exactly one of (peerType+peerTarget), contactID, or addressID must be non-zero.
// Returns peer_events (peerevent.PeerEvent) unmodified — no reshaping (design doc §8.1/§9).
func (h *serviceHandler) InteractionList(
	ctx context.Context,
	a *auth.AuthIdentity,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
) ([]*tmpeerevent.PeerEvent, string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "InteractionList",
		"customer_id": a.CustomerID,
		"peer_type":   peerType,
		"peer_target": peerTarget,
		"contact_id":  contactID,
		"address_id":  addressID,
	})

	if a.IsDirect() {
		return nil, "", serviceerrors.ErrDirectAccessNotSupported
	}

	if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
		return nil, "", serviceerrors.ErrPermissionDenied
	}

	items, nextToken, err := h.reqHandler.ContactV1InteractionList(ctx, a.CustomerID, size, token, peerType, peerTarget, contactID, addressID, time.Time{})
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		return nil, "", err
	}

	return items, nextToken, nil
}
