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
)

// processV1RecordingGet handles GET /v1/recordings request
func (h *listenHandler) processV1RecordingsGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

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
		"customer_id": customerID,
		"size":        pageSize,
		"token":       pageToken,
	})

	log.Debug("Getting recordings.")
	tmps, err := h.recordingHandler.GetsByCustomerID(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get recordings. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmps)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmps, err)
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

// processV1RecordingsIDGet handles GET /v1/recordings/<id> request
func (h *listenHandler) processV1RecordingsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDGet.")

	tmp, err := h.recordingHandler.Get(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.Debugf("Found record. record: %v", tmp)

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

// processV1RecordingsIDDelete handles DELETE /v1/recordings/<id> request
func (h *listenHandler) processV1RecordingsIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1RecordingsIDDelete.")

	tmp, err := h.recordingHandler.Delete(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.Debugf("Found record. record: %v", tmp)

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
