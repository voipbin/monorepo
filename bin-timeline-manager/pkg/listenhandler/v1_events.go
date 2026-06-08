package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

func (h *listenHandler) v1EventsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1EventsPost",
	})

	// Parse request
	var req request.V1DataEventsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Call handler with the unwrapped domain inputs. The listenhandler is the
	// single layer that touches request.* / response.* transport DTOs.
	res, err := h.eventHandler.List(ctx, req.Publisher, req.ResourceID, req.Events, req.PageToken, req.PageSize)
	if err != nil {
		log.Errorf("Could not list events. err: %v", err)
		return errorResponse(err), nil
	}

	// Map the domain result into the transport DTO. The listenhandler is the
	// single layer that constructs response.* types.
	result := &response.V1DataEventsPost{
		Result:        res.Result,
		NextPageToken: res.NextPageToken,
	}

	// Marshal response
	data, err := json.Marshal(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal response")
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
