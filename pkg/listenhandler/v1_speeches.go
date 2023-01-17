package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/request"
)

// v1SpeechesPost handles /v1/speeches POST request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1SpeechesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithField(
		"func", "v1SpeechesPost",
	)

	var req request.V1DataSpeechesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}
	log.WithField("request", req).Debugf("Request detail.")

	// create tts
	tmp, err := h.ttshandler.Create(ctx, req.CallID, req.Text, req.Language, req.Gender)
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
