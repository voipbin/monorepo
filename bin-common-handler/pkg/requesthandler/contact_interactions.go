package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/sock"

	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
)

// ContactV1InteractionList lists interactions from contact-manager.
// Exactly one of (peerType+peerTarget), contactID, or addressID should be non-zero,
// UNLESS since is non-zero, in which case zero filters is allowed (unfiltered, time-scoped mode).
// since is encoded as an absolute RFC3339Nano timestamp on the wire (not a relative "Nd" string) —
// any "Nd" parsing/range validation happens at the bin-api-manager HTTP layer before this call.
//
// The RPC's internal implementation proxies bin-timeline-manager's peer_events read API
// (design doc 2026-07-25-contact-interaction-retire-to-peer-events, §8.1/§9), so the
// response shape is peerevent.PeerEvent, returned without reshaping.
func (r *requestHandler) ContactV1InteractionList(
	ctx context.Context,
	customerID uuid.UUID,
	size uint64,
	token string,
	peerType, peerTarget string,
	contactID, addressID uuid.UUID,
	since time.Time,
) ([]*tmpeerevent.PeerEvent, string, error) {
	u := url.Values{}
	if peerType != "" {
		u.Set("peer_type", peerType)
	}
	if peerTarget != "" {
		u.Set("peer_target", peerTarget)
	}
	if contactID != uuid.Nil {
		u.Set("contact_id", contactID.String())
	}
	if addressID != uuid.Nil {
		u.Set("address_id", addressID.String())
	}
	if !since.IsZero() {
		u.Set("since", since.UTC().Format(time.RFC3339Nano))
	}
	if size > 0 {
		u.Set("page_size", fmt.Sprintf("%d", size))
	}
	if token != "" {
		u.Set("page_token", token)
	}

	uri := "/v1/interactions?" + u.Encode()

	m, err := json.Marshal(map[string]string{"customer_id": customerID.String()})
	if err != nil {
		return nil, "", err
	}

	tmp, err := r.sendRequestContact(ctx, uri, sock.RequestMethodGet, "contact/interactions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, "", err
	}

	var res []*tmpeerevent.PeerEvent
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, "", errParse
	}

	nextToken := ""
	if len(res) > 0 {
		nextToken = res[len(res)-1].Timestamp.UTC().Format("2006-01-02T15:04:05.000000Z")
	}

	return res, nextToken, nil
}
