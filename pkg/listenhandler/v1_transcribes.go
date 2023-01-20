package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// processV1TranscribesPost handles POST /v1/transcribes request
// It creates a new transcribe.
func (h *listenHandler) processV1TranscribesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1TranscribesPost",
		})
	log.WithField("request", m).Debug("Executing processV1TranscribesPost.")

	var req request.V1DataTranscribesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"customer_id":    req.CustomerID,
		"reference_type": req.ReferenceType,
		"reference_id":   req.ReferenceID,
	})

	log.Debug("Starting transcribe.")
	tmp, err := h.transcribeHandler.Start(ctx, req.CustomerID, req.ReferenceType, req.ReferenceID, req.Language, req.Direction)
	if err != nil {
		log.Debugf("Could not create a transcribe. err: %v", err)
		return simpleResponse(500), nil
	}
	log.WithField("transcribe", tmp).Debugf("Created transcribe. transcribe_id: %s", tmp.ID)

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TranscribesGet handles GET /v1/transcribes request
func (h *listenHandler) processV1TranscribesGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get customer_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	log := logrus.WithFields(logrus.Fields{
		"user":  customerID,
		"size":  pageSize,
		"token": pageToken,
	})

	calls, err := h.transcribeHandler.Gets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(calls)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", calls, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1TranscribesIDGet handles GET /v1/transcribes/<id> request
func (h *listenHandler) processV1TranscribesIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1TranscribesIDGet.")

	c, err := h.transcribeHandler.Get(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
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

// processV1TranscribesIDDelete handles Delete /v1/transcribes/<id> request
func (h *listenHandler) processV1TranscribesIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1TranscribesIDDelete.")

	tmp, err := h.transcribeHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the transcribe. err: %v", err)
		return simpleResponse(400), nil
	}

	d, err := json.Marshal(tmp)
	if err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       d,
	}

	return res, nil
}

// processV1TranscribesIDStopPost handles /v1/transcribes/<id>/stop POST request
func (h *listenHandler) processV1TranscribesIDStopPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"handler": "processV1TranscribesIDStopPost",
			"uri":     m.URI,
		},
	)

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tr, err := h.transcribeHandler.Stop(ctx, id)
	if err != nil {
		log.Errorf("Could not update the conference. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := json.Marshal(tr)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       tmp,
	}

	return res, nil
}
