package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
)

// processV1InteractionsGet handles GET /v1/interactions?... request.
// Exactly one of peer_type+peer_target, contact_id, or address_id is required.
// customer_id is read from the JSON request body (consistent with other GET handlers).
//
// The response is proxied from bin-timeline-manager's peer_events read API
// (design doc 2026-07-25-contact-interaction-retire-to-peer-events, §8.1/§9):
// contacthandler.InteractionList returns []*peerevent.PeerEvent, which is
// serialized here without reshaping.
func (h *listenHandler) processV1InteractionsGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1InteractionsGet",
	})
	log.WithField("request", req).Debug("Received request.")

	u, err := url.Parse(req.URI)
	if err != nil {
		return simpleResponse(400), nil
	}
	q := u.Query()

	// Parse pagination params from query string.
	tmpSize, _ := strconv.Atoi(q.Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := q.Get(PageToken)

	// Parse filter params.
	peerType := q.Get("peer_type")
	peerTarget := q.Get("peer_target")

	var contactID uuid.UUID
	if s := q.Get("contact_id"); s != "" {
		contactID = uuid.FromStringOrNil(s)
	}
	var addressID uuid.UUID
	if s := q.Get("address_id"); s != "" {
		addressID = uuid.FromStringOrNil(s)
	}

	// customer_id from JSON request body.
	var customerID uuid.UUID
	if len(req.Data) > 0 {
		var bodyMap map[string]interface{}
		if jsonErr := json.Unmarshal(req.Data, &bodyMap); jsonErr == nil {
			if v, ok := bodyMap["customer_id"].(string); ok {
				customerID = uuid.FromStringOrNil(v)
			}
		}
	}
	if customerID == uuid.Nil {
		log.Error("Missing customer_id in request body.")
		return simpleResponse(400), nil
	}

	// Validate: exactly one filter mode is required.
	filterCount := 0
	if peerType != "" || peerTarget != "" {
		filterCount++
	}
	if contactID != uuid.Nil {
		filterCount++
	}
	if addressID != uuid.Nil {
		filterCount++
	}
	if filterCount != 1 {
		log.Errorf("Expected exactly one filter mode, got %d.", filterCount)
		return simpleResponse(400), nil
	}

	res, _, err := h.contactHandler.InteractionList(ctx, customerID, pageSize, pageToken, peerType, peerTarget, contactID, addressID)
	if err != nil {
		log.Errorf("Could not list interactions. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
