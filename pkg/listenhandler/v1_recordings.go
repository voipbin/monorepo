package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1RecordingGet handles GET /v1/recordings request
func (h *listenHandler) processV1RecordingsGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var reqData request.V1DataRecordingsGET
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log := logrus.WithFields(logrus.Fields{
		"user":  reqData.UserID,
		"size":  reqData.PageSize,
		"token": reqData.PageToken,
	})

	log.Debug("Getting recordings.")
	recordings, err := h.db.RecordingGets(context.Background(), reqData.UserID, reqData.PageSize, reqData.PageToken)
	if err != nil {
		log.Debugf("Could not get recordings. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(recordings)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", recordings, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1RecordingsIDGet handles GET /v1/recordings/<id> request
func (h *listenHandler) processV1RecordingsIDGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id, err := url.QueryUnescape(uriItems[3])
	if err != nil {
		logrus.Errorf("Could not decode the id. err: %v", err)
		return simpleResponse(404), nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1CallsIDGet.")

	record, err := h.db.RecordingGet(ctx, id)
	if err != nil {
		return simpleResponse(404), nil
	}
	log.Debugf("Found record. record: %v", record)

	data, err := json.Marshal(record)
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
