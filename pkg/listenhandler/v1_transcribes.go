package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1TranscribesIDDelete handles Delete /v1/transcribes/<id> request
func (h *listenHandler) processV1TranscribesIDDelete(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	ctx := context.Background()
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
