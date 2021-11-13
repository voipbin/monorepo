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

// processV1ConfbridgesPost handles /v1/confbriges request
func (h *listenHandler) processV1ConfbridgesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConfbridgesPost",
			"uri":     m.URI,
			"data":    m.Data,
		},
	)

	var data request.V1DataConfbridgesPost
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	// create confbridge
	cb, err := h.confbridgeHandler.Create(ctx, data.ConferenceID)

	if err != nil {
		log.Errorf("Could not create the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cb)
	if err != nil {
		log.Errorf("Could not marshal the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		Data:       tmp,
	}

	return res, nil
}

// processV1ConfbridgesIDGet handles /v1/confbriges/<id> Get request
func (h *listenHandler) processV1ConfbridgesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.WithField("request", m).Debug("Executing processV1ConfbridgesIDGet.")

	// create confbridge
	cb, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cb)
	if err != nil {
		log.Errorf("Could not marshal the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		Data:       tmp,
	}

	return res, nil
}

// processV1ConfbridgesIDDelete handles /v1/confbridges/<id> DELETE request
func (h *listenHandler) processV1ConfbridgesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConfbridgesIDDelete",
			"uri":     m.URI,
		},
	)
	log.Debugf("Deleting confbridge. request: %v", m)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	if err := h.confbridgeHandler.Terminate(ctx, id); err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}

// processV1ConfbridgesIDCallsIDDelete handles /v1/confbridges/<confbridge-id>/calls/<call-id> DELETE request
func (h *listenHandler) processV1ConfbridgesIDCallsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConfbridgesIDCallsIDDelete",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	callID := uuid.FromStringOrNil(uriItems[5])

	if err := h.confbridgeHandler.Kick(ctx, id, callID); err != nil {
		log.Errorf("Could not kick out the call from the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}

// processV1ConfbridgesIDCallsIDPost handles /v1/confbridges/<confbridge-id>/calls/<call-id> DELETE request
func (h *listenHandler) processV1ConfbridgesIDCallsIDPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1ConfbridgesIDCallsIDDelete",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 6 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])
	callID := uuid.FromStringOrNil(uriItems[5])

	if err := h.confbridgeHandler.Join(ctx, id, callID); err != nil {
		log.Errorf("Could not join the call to the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}
