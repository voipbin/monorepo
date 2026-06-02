package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1QueuesIDDirectHashRegeneratePost handles POST /v1/queues/<queue-id>/direct-hash-regenerate request
func (h *listenHandler) processV1QueuesIDDirectHashRegeneratePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":     "processV1QueuesIDDirectHashRegeneratePost",
		"queue_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	tmp, err := h.queueHandler.DirectHashRegenerate(ctx, id)
	if err != nil {
		log.Errorf("Could not regenerate direct hash. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
