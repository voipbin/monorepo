package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// v1CorrelationsGet handles GET /v1/correlations/<resource_id> request.
// Returns the correlation graph of all resources sharing the resource's activeflow.
func (h *listenHandler) v1CorrelationsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1CorrelationsGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	resourceID := uuid.FromStringOrNil(uriItems[3])
	if resourceID == uuid.Nil {
		return simpleResponse(400), nil
	}

	res, err := h.eventHandler.ResourceCorrelationGet(ctx, resourceID)
	if err != nil {
		log.Errorf("Could not get resource correlation. err: %v", err)
		return errorResponse(err), nil
	}

	// Map the domain result into the transport DTO. The listenhandler is the
	// single layer that constructs response.* types.
	result := &response.V1DataResourceCorrelationGet{
		ResourceID:    res.ResourceID,
		ResourceFound: res.ResourceFound,
		ActiveflowID:  res.ActiveflowID,
		Truncated:     res.Truncated,
		Resources:     res.Resources,
	}

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
