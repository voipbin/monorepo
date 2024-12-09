package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/externalmedia"
	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
)

// processV1ConfbridgesPost handles /v1/confbriges request
func (h *listenHandler) processV1ConfbridgesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesPost",
		"request": m,
	})

	var req request.V1DataConfbridgesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id": req.CustomerID,
		"type":        req.Type,
	})

	// create confbridge
	cb, err := h.confbridgeHandler.Create(ctx, req.CustomerID, req.Type)
	if err != nil {
		log.Errorf("Could not create the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cb)
	if err != nil {
		log.Errorf("Could not marshal the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}

// processV1ConfbridgesIDGet handles /v1/confbriges/<id> Get request
func (h *listenHandler) processV1ConfbridgesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	cb, err := h.confbridgeHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(cb)
	if err != nil {
		log.Errorf("Could not marshal the confbridge. err: %v", err)
		return simpleResponse(400), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		Data:       tmp,
	}

	return res, nil
}

// processV1ConfbridgesIDDelete handles /v1/confbridges/<id> DELETE request
func (h *listenHandler) processV1ConfbridgesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.confbridgeHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return simpleResponse(400), nil
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

// processV1ConfbridgesIDTerminatePost handles /v1/confbridges/<id>/terminate POST request
func (h *listenHandler) processV1ConfbridgesIDTerminatePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDTerminatePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.confbridgeHandler.Terminating(ctx, id)
	if err != nil {
		log.Errorf("Could not terminate the confbridge. err: %v", err)
		return simpleResponse(400), nil
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

// processV1ConfbridgesIDCallsIDDelete handles /v1/confbridges/<confbridge-id>/calls/<call-id> DELETE request
func (h *listenHandler) processV1ConfbridgesIDCallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDCallsIDDelete",
		"request": m,
	})

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
func (h *listenHandler) processV1ConfbridgesIDCallsIDPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDCallsIDPost",
		"request": m,
	})

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

// processV1ConfbridgesIDExternalMediaPost handles /v1/confbridges/<confbridge-id>/external-media POST request
func (h *listenHandler) processV1ConfbridgesIDExternalMediaPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDExternalMediaPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConfbridgesIDExternalMediaPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.confbridgeHandler.ExternalMediaStart(ctx, id, req.ExternalMediaID, req.ExternalHost, externalmedia.Encapsulation(req.Encapsulation), externalmedia.Transport(req.Transport), req.ConnectionType, req.Format, req.Direction)
	if err != nil {
		log.Errorf("Could not start the external media. confbridge_id: %s, err: %v", id, err)
		return nil, err
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

// processV1ConfbridgesIDExternalMediaDelete handles /v1/confbridges/<confbridge-id>/external-media DELETE request
func (h *listenHandler) processV1ConfbridgesIDExternalMediaDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDExternalMediaDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.confbridgeHandler.ExternalMediaStop(ctx, id)
	if err != nil {
		log.Errorf("Could not stop the external media. confbridge_id: %s, err: %v", id, err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConfbridgesIDRecordingStartPost handles /v1/confbridges/<confbridge-id>/recording_start POST request
func (h *listenHandler) processV1ConfbridgesIDRecordingStartPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDRecordingStartPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConfbridgesIDRecordingStartPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.confbridgeHandler.RecordingStart(ctx, id, req.Format, req.EndOfSilence, req.EndOfKey, req.Duration)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConfbridgesIDRecordingStopPost handles /v1/confbridges/<confbridge-id>/recording_stop POST request
func (h *listenHandler) processV1ConfbridgesIDRecordingStopPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDRecordingStopPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.confbridgeHandler.RecordingStop(ctx, id)
	if err != nil {
		log.Errorf("Could not start call recording. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConfbridgesIDFlagsPost handles /v1/confbridges/<confbridge-id>/flags POST request
func (h *listenHandler) processV1ConfbridgesIDFlagsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDFlagsPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConfbridgesIDFlagsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.confbridgeHandler.FlagAdd(ctx, id, req.Flag)
	if err != nil {
		log.Errorf("Could not add the flag. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConfbridgesIDFlagsDelete handles /v1/confbridges/<confbridge-id>/flags DELETE request
func (h *listenHandler) processV1ConfbridgesIDFlagsDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDFlagsDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataConfbridgesIDFlagsDelete
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.confbridgeHandler.FlagRemove(ctx, id, req.Flag)
	if err != nil {
		log.Errorf("Could not remove the flag. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", data, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConfbridgesIDRingPost handles /v1/confbridges/<confbridge-id>/ring POST request
func (h *listenHandler) processV1ConfbridgesIDRingPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDRingPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errRing := h.confbridgeHandler.Ring(ctx, id); errRing != nil {
		log.Errorf("Could not ring the confbridge. err: %v", errRing)
		return simpleResponse(500), nil
	}

	return simpleResponse(200), nil
}

// processV1ConfbridgesIDAnswerPost handles /v1/confbridges/<confbridge-id>/answer POST request
func (h *listenHandler) processV1ConfbridgesIDAnswerPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConfbridgesIDAnswerPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	if errRing := h.confbridgeHandler.Answer(ctx, id); errRing != nil {
		log.Errorf("Could not answer the confbridge. err: %v", errRing)
		return simpleResponse(500), nil
	}

	return simpleResponse(200), nil
}
