package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/response"
)

// processV1StreamingsPost handles POST /v1/streamings request
// It creates a new speech-to-text.
func (h *listenHandler) processV1StreamingsPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 3 {
		return simpleResponse(400), nil
	}

	var reqData request.V1DataStreamingsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		logrus.Errorf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// do transcribe
	ctx := context.Background()
	trans, err := h.transcribeHandler.StreamingTranscribeStart(
		ctx, reqData.UserID, reqData.ReferenceID, transcribe.Type(reqData.Type), reqData.Language, reqData.WebhookURI, reqData.WebhookMethod)
	if err != nil {
		return simpleResponse(400), nil
	}

	s := &response.V1ResponseStreamingsPost{
		Transcribe: trans,
	}

	d, err := json.Marshal(s)
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

// processV1StreamingsPost handles Delete /v1/streamings/<id> request
// It stops speech-to-text.
func (h *listenHandler) processV1StreamingsIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debug("Executing processV1StreamingsIDDelete.")

	ctx := context.Background()
	if err := h.transcribeHandler.StreamingTranscribeStop(ctx, id); err != nil {
		log.Errorf("Could not stop the transcribe. err: %v", err)
		return simpleResponse(400), nil
	}

	var reqData request.V1DataStreamingsPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		logrus.Errorf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	return simpleResponse(200), nil
}
