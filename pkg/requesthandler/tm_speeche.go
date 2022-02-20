package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/tts-manager.git/pkg/listenhandler/models/response"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// TMV1SpeecheCreate create speech-to-text.
func (r *requestHandler) TMV1SpeecheCreate(ctx context.Context, callID uuid.UUID, text, gender, language string, timeout int) (string, error) {

	uri := "/v1/speeches"

	m, err := json.Marshal(request.V1DataSpeechesPost{
		CallID:   callID,
		Text:     text,
		Gender:   gender,
		Language: language,
	})
	if err != nil {
		return "", err
	}

	res, err := r.sendRequestTTS(uri, rabbitmqhandler.RequestMethodPost, resourceTTSSpeeches, timeout, 0, ContentTypeJSON, m)
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
