package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// TMStreamingsPost sends a request to transcribe-manager
// to start the streaming transcribe
func (r *requestHandler) TMStreamingsPost(callID uuid.UUID, language, webhookURI, webhookMethod string) (*transcribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/streamings")

	data := &tmrequest.V1DataStreamingsPost{
		ReferenceID:   callID,
		Type:          string(transcribe.TypeCall),
		Language:      language,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestTranscribe(uri, rabbitmqhandler.RequestMethodPost, resourceTranscribeStreamings, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res transcribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
