package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/response"
)

func (r *requestHandler) TTSSpeechesPOST(text, gender, language string) (string, error) {

	uri := "/v1/speeches"

	m, err := json.Marshal(request.V1DataSpeechesPost{
		Text:     text,
		Gender:   gender,
		Language: language,
	})
	if err != nil {
		return "", err
	}

	res, err := r.sendRequestTTS(uri, rabbitmqhandler.RequestMethodPost, resourceTTSSpeeches, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return "", err
	}

	if res.StatusCode >= 299 {
		return "", fmt.Errorf("could not find action")
	}

	var resData response.V1ResponseSpeechesPost
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return "", err
	}

	return resData.Filename, nil
}
