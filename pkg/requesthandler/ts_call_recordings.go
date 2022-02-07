package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// TSV1CallRecordingCreate sends a request to transcribe-manager
// to transcribe call-recording.
func (r *requestHandler) TSV1CallRecordingCreate(ctx context.Context, customerID, callID uuid.UUID, language string, timeout, delay int) ([]tstranscribe.Transcribe, error) {
	uri := "/v1/call_recordings"

	req := &tmrequest.V1DataCallRecordingsPost{
		CustomerID:  customerID,
		ReferenceID: callID,
		Language:    language,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestTS(uri, rabbitmqhandler.RequestMethodPost, resourceTranscribeCallRecordings, timeout, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data []tstranscribe.Transcribe
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return data, nil

}
