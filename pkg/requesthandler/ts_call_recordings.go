package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tmrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// TSV1CallRecordingCreate sends a request to transcribe-manager
// to transcribe call-recording.
func (r *requestHandler) TSV1CallRecordingCreate(ctx context.Context, callID uuid.UUID, language, webhookURI, webhookMethod string, timeout, delay int) error {
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

	tmp, err := r.sendRequestTS(uri, rabbitmqhandler.RequestMethodPost, resourceTranscribeCallRecordings, timeout, delay, ContentTypeJSON, m)
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
