package requesthandler

import (
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler/models/request"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler/models/response"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func (r *requestHandler) TTSSpeechesPOST(text, gender, language string) (string, error) {

	uri := fmt.Sprintf("/v1/speeches")

	m, err := json.Marshal(request.TTSV1DataSpeechesPost{
		Text:     text,
		Gender:   gender,
		Language: language,
	})
	if err != nil {
		return "", err
	}

	res, err := r.sendRequestTTS(uri, rabbitmqhandler.RequestMethodPost, resourceTTSSpeeches, requestTimeoutDefault, ContentTypeJSON, m)
	if err != nil {
		return "", err
	}

	if res.StatusCode >= 299 {
		return "", fmt.Errorf("could not find action")
	}

	var resData response.TTSV1ResponseSpeechesPost
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return "", err
	}

	return resData.URL, nil
}
