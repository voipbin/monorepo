package listenhandler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// v1PeerEventsGet handles GET /v1/peer-events — list peer_events rows
// matching the given peer addresses, scoped to customer_id.
// customer_id/page_token/page_size arrive as query params (the
// requesthandler authority, same split v1AnalysesGet uses); peer_addresses
// arrives as a JSON body (an array, same reason /v1/events keeps its
// `events []string` filter in the body rather than the query string).
func (h *listenHandler) v1PeerEventsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithField("func", "v1PeerEventsGet")

	if h.peerEventHandler == nil {
		return simpleResponse(http.StatusServiceUnavailable), nil
	}

	q := queryValues(m.URI)
	customerID := uuid.FromStringOrNil(q.Get("customer_id"))
	if customerID == uuid.Nil {
		return simpleResponse(http.StatusBadRequest), nil
	}
	pageToken := q.Get("page_token")
	pageSize := parsePageSize(q.Get("page_size"))

	var req request.V1DataPeerEventsGet
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &req); err != nil {
			log.Errorf("Could not unmarshal request. err: %v", err)
			return simpleResponse(http.StatusBadRequest), nil
		}
	}

	res, err := h.peerEventHandler.List(ctx, customerID, req.PeerAddresses, pageToken, int(pageSize))
	if err != nil {
		log.Errorf("Could not list peer events. err: %v", err)
		return errorResponse(err), nil
	}

	result := &response.V1DataPeerEventsGet{
		Result:        res.Result,
		NextPageToken: res.NextPageToken,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal response")
	}
	return &sock.Response{StatusCode: 200, DataType: "application/json", Data: data}, nil
}
