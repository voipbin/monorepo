package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1AIsIDActivateInsightPost handles POST /v1/ais/<ai-id>/activate_insight request
func (h *listenHandler) processV1AIsIDActivateInsightPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":  "processV1AIsIDActivateInsightPost",
		"ai_id": id,
	})
	log.WithField("request", m).Debug("Received request.")

	if id == uuid.Nil {
		log.Error("Invalid AI ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.aiHandler.ActivateInsight(ctx, id)
	if err != nil {
		log.Errorf("Could not activate the insight ai. err: %v", err)
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
