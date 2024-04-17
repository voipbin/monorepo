package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/pkg/listenhandler/models/request"
)

// v1SpeechesPost handles /v1/speeches POST request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1SpeechesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SpeechesPost",
	})

	var req request.V1DataSpeechesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Request detail.")

	// create tts
	tmp, err := h.ttsHandler.Create(ctx, req.CallID, req.Text, req.Language, req.Gender)
	if err != nil {
		log.Errorf("Could not create a tts audio. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
