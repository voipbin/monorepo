package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-pipecat-manager/pkg/listenhandler/models/request"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1PipecatcallsPost handles POST /v1/pipecatcalls request
// It creates a new pipecatcall.
func (h *listenHandler) processV1PipecatcallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1PipecatcallsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var req request.V1DataPipecatcallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.pipecatcallHandler.Start(
		ctx,
		req.CustomerID,
		req.ActiveflowID,
		req.ReferenceType,
		req.ReferenceID,
		req.LLM,
		req.STT,
		req.TTS,
		req.VoiceID,
		req.Messages,
	)
	if err != nil {
		log.Debugf("Could not create a pipecatcall. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1PipecatcallsIDGet handles GET /v1/pipecatcalls/<id> request
func (h *listenHandler) processV1PipecatcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1PipecatcallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.pipecatcallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get resource. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1PipecatcallsIDStopPost handles /v1/pipecatcalls/<id>/stop POST request
func (h *listenHandler) processV1PipecatcallsIDStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1PipecatcallsIDStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.pipecatcallHandler.Stop(ctx, id)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
