package listenhandler

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/listenhandler/models/request"
)

// processV1RecordingsPost handles POST /v1/recordings request
// It creates a new speech-to-text.
func (h *listenHandler) processV1RecordingsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataRecordingsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// do stt
	if err := h.sttHandler.Recording(reqData.ReferenceID, reqData.Language, reqData.WebhookURI, reqData.WebhookMethod); err != nil {
		logrus.Debugf("Could not create stt. err: %v", err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	return res, nil
}
