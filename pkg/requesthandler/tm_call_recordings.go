package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tmrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// SMCallRecordingPost sends a request to number-manager
// to delete a flow from the number.
func (r *requestHandler) TMCallRecordingPost(callID uuid.UUID, language, webhookURI, webhookMethod string, timeout, delay int) error {
	uri := "/v1/call_recordings"

	data := &tmrequest.V1DataCallRecordingsPost{
		ReferenceID:   callID,
		Language:      language,
		WebhookURI:    webhookURI,
		WebhookMethod: webhookMethod,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestTranscribe(uri, rabbitmqhandler.RequestMethodPost, resourceTranscribeCallRecordings, timeout, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
