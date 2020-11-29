package listenhandler

import (
	"encoding/json"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/response"
)

// v1SpeechesPost handles /v1/speeches POST request
// creates a new tts audio for the given text and upload the file to the bucket. Returns uploaded filename with path.
func (h *listenHandler) v1SpeechesPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	var reqData request.V1DataSpeechesPost
	if err := json.Unmarshal(req.Data, &reqData); err != nil {
		logrus.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	// create tts
	url, err := h.ttshandler.TTSCreate(reqData.Text, reqData.Language, reqData.Gender)
	if err != nil {
		logrus.Errorf("Could not create a tts audio. err: %v", err)
		return nil, err
	}

	resMsg := &response.V1ResponseSpeechesPost{
		URL: url,
	}

	data, err := json.Marshal(resMsg)
	if err != nil {
		logrus.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
