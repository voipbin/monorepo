package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

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
