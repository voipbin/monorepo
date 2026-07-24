package contacthandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/pkg/dbhandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// InteractionList is the main timeline read path. It proxies
// bin-timeline-manager's peer_events read API (design doc
// 2026-07-25-contact-interaction-retire-to-peer-events, §8.1/§9).
//
// Exactly one of (peerType+peerTarget), contactID, or addressID must be
// non-zero -- peer_events requires at least one address filter, unlike the
// old contact_interactions InteractionList, which additionally supported an
// unfiltered/since-only mode. That mode has no peer_events equivalent and is
// rejected here.
//
// Returns the response without reshaping (§8.1 item 1 / §9.1 item 1):
// []*peerevent.PeerEvent, plus a next-page token (empty when no further pages).
func (h *contactHandler) InteractionList(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID uuid.UUID,
	addressID uuid.UUID,
	since time.Time,
) ([]*tmpeerevent.PeerEvent, string, error) {
	var addrs []commonaddress.Address

	switch {
	case peerType != "" || peerTarget != "":
		addrs = []commonaddress.Address{
			{
				Type:   commonaddress.Type(peerType),
				Target: peerTarget,
			},
		}

	case contactID != uuid.Nil:
		c, err := h.db.ContactGet(ctx, contactID)
		if err != nil || c == nil || c.TMDelete != nil || c.CustomerID != customerID {
			return nil, "", cerrors.NotFound(
				commonoutline.ServiceNameContactManager,
				"CONTACT_NOT_FOUND",
				"The contact was not found.",
			)
		}
		for _, a := range c.Addresses {
			addrs = append(addrs, a.Address)
		}

	case addressID != uuid.Nil:
		ap, err := h.db.AddressGet(ctx, customerID, addressID)
		if err != nil {
			if stderrors.Is(err, dbhandler.ErrNotFound) {
				return nil, "", cerrors.NotFound(
					commonoutline.ServiceNameContactManager,
					"ADDRESS_NOT_FOUND",
					"The address was not found.",
				)
			}
			return nil, "", fmt.Errorf("could not get address. InteractionList. err: %v", err)
		}
		addrs = []commonaddress.Address{ap.Address}

	default:
		return nil, "", cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"INVALID_FILTER",
			"At least one filter (peer_type+peer_target, contact_id, or address_id) is required.",
		)
	}

	if len(addrs) == 0 {
		return nil, "", cerrors.InvalidArgument(
			commonoutline.ServiceNameContactManager,
			"INVALID_FILTER",
			"At least one filter (peer_type+peer_target, contact_id, or address_id) is required.",
		)
	}

	req := &tmpeerevent.PeerEventListRequest{
		CustomerID:    customerID,
		PeerAddresses: addrs,
		PageToken:     token,
		PageSize:      int(size),
	}

	res, err := h.reqHandler.TimelineV1PeerEventList(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("could not list peer events. InteractionList. err: %v", err)
	}

	return res.Result, res.NextPageToken, nil
}
