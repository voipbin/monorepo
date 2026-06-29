package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-contact-manager/pkg/listenhandler/models/request"
)

// parseDaysDuration parses a "Nd" duration string (e.g. "7d", "30d") into time.Duration.
// Returns an error for any other format, negative values, or values exceeding maxDays.
func parseDaysDuration(s string, maxDays int) (time.Duration, error) {
	if !strings.HasSuffix(s, "d") {
		return 0, fmt.Errorf("invalid duration format %q: must be '<N>d' (e.g. '7d')", s)
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
	if err != nil {
		return 0, fmt.Errorf("invalid duration format %q: %w", s, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid duration %q: must be a positive number of days (e.g. '7d')", s)
	}
	if n > maxDays {
		return 0, fmt.Errorf("duration %q exceeds maximum of %dd", s, maxDays)
	}
	return time.Duration(n) * 24 * time.Hour, nil
}

// processV1InteractionsGet handles GET /v1/interactions?... request.
// Exactly one of peer_type+peer_target, contact_id, or address_id is required.
// customer_id is read from the JSON request body (consistent with other GET handlers).
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

	// Validate: exactly one filter mode.
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

// processV1InteractionsUnresolvedGet handles GET /v1/interactions/unresolved request.
func (h *listenHandler) processV1InteractionsUnresolvedGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1InteractionsUnresolvedGet",
	})
	log.WithField("request", req).Debug("Received request.")

	u, err := url.Parse(req.URI)
	if err != nil {
		return simpleResponse(400), nil
	}
	q := u.Query()

	tmpSize, _ := strconv.Atoi(q.Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := q.Get(PageToken)

	// Parse since (default 30d, max 180d).
	sinceStr := q.Get("since")
	if sinceStr == "" {
		sinceStr = "30d"
	}
	sinceDuration, parseErr := parseDaysDuration(sinceStr, 180)
	if parseErr != nil {
		log.Errorf("Invalid since param: %v", parseErr)
		return simpleResponse(400), nil
	}
	since := time.Now().Add(-sinceDuration)

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

	res, _, err := h.contactHandler.InteractionListUnresolved(ctx, customerID, pageSize, pageToken, since)
	if err != nil {
		log.Errorf("Could not list unresolved interactions. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1InteractionsIDGet handles GET /v1/interactions/{id} request.
func (h *listenHandler) processV1InteractionsIDGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1InteractionsIDGet",
	})
	log.WithField("request", req).Debug("Received request.")

	// Extract id from URI.
	parts := strings.Split(req.URI, "/")
	// URI pattern: /v1/interactions/<uuid>
	if len(parts) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(parts[3])
	if id == uuid.Nil {
		return simpleResponse(400), nil
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

	res, err := h.contactHandler.InteractionGet(ctx, customerID, id)
	if err != nil {
		log.Errorf("Could not get interaction. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1InteractionsResolutionsPost handles POST /v1/interactions/{id}/resolutions request.
func (h *listenHandler) processV1InteractionsResolutionsPost(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1InteractionsResolutionsPost",
	})
	log.WithField("request", req).Debug("Received request.")

	// Extract interaction id from URI: /v1/interactions/<uuid>/resolutions
	parts := strings.Split(req.URI, "/")
	if len(parts) < 4 {
		return simpleResponse(400), nil
	}
	interactionID := uuid.FromStringOrNil(parts[3])
	if interactionID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataInteractionsResolutionsPost
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}

	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.contactHandler.ResolutionCreate(ctx, body.CustomerID, body.ContactID, interactionID,
		body.ResolutionType, body.ResolvedByType, body.ResolvedByID)
	if err != nil {
		log.Errorf("Could not create resolution. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		return simpleResponse(500), nil
	}

	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}

// processV1InteractionsResolutionsIDDelete handles DELETE /v1/interactions/{id}/resolutions/{rid} request.
func (h *listenHandler) processV1InteractionsResolutionsIDDelete(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "processV1InteractionsResolutionsIDDelete",
	})
	log.WithField("request", req).Debug("Received request.")

	// URI: /v1/interactions/<uuid>/resolutions/<uuid>
	parts := strings.Split(req.URI, "/")
	if len(parts) < 6 {
		return simpleResponse(400), nil
	}
	interactionID := uuid.FromStringOrNil(parts[3])
	if interactionID == uuid.Nil {
		return simpleResponse(400), nil
	}
	resolutionID := uuid.FromStringOrNil(parts[5])
	if resolutionID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var body request.V1DataInteractionsResolutionsIDDelete
	if err := json.Unmarshal(req.Data, &body); err != nil {
		log.Errorf("Could not unmarshal request body. err: %v", err)
		return simpleResponse(400), nil
	}

	if body.CustomerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	if err := h.contactHandler.ResolutionDelete(ctx, body.CustomerID, interactionID, resolutionID); err != nil {
		log.Errorf("Could not delete resolution. err: %v", err)
		return errorResponse(err), nil
	}

	return simpleResponse(200), nil
}
