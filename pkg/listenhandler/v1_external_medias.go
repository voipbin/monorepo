package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
)

// processV1ExternalMediasPost handles POST /v1/external-medias request
func (h *listenHandler) processV1ExternalMediasPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ExternalMediasPost",
			"uri":     m.URI,
			"data":    m.Data,
		},
	)

	var req request.V1DataExternalMediasPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.externalMediaHandler.Start(
		ctx,
		req.ReferenceType,
		req.ReferenceID,
		req.ExternalHost,
		req.Encapsulation,
		req.Transport,
		req.ConnectionType,
		req.Format,
		req.Direction,
	)
	if err != nil {
		log.Errorf("Could not start the external media. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExternalMediasIDGet handles GET /v1/external-medias/<external-media-id> request
func (h *listenHandler) processV1ExternalMediasIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1ExternalMediasIDGet.")

	tmp, err := h.externalMediaHandler.Get(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.WithField("external_media", tmp).Debugf("Found external_media. external_media_id: %s", tmp.ID)

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ExternalMediasIDDelete handles DELETE /v1/external-medias/<id> request
func (h *listenHandler) processV1ExternalMediasIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1ExternalMediasIDDelete.")

	tmp, err := h.externalMediaHandler.Stop(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.WithField("external_media", tmp).Debugf("Stopped external media. external_media_id: %s", tmp.ID)

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
