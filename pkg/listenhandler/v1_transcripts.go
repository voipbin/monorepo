package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processV1TranscriptsGet handles GET /v1/transcripts request
func (h *listenHandler) processV1TranscriptsGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	transcribeID := uuid.FromStringOrNil(u.Query().Get("transcribe_id"))
	log := logrus.WithFields(logrus.Fields{
		"transcribe_id": transcribeID,
	})

	tmp, err := h.transcriptHandler.Gets(ctx, transcribeID)
	if err != nil {
		log.Errorf("Could not get transcribes. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
