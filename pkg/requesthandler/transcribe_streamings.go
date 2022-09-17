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

// TranscribeV1StreamingCreate sends a request to transcribe-manager
// to start the streaming transcribe
func (r *requestHandler) TranscribeV1StreamingCreate(ctx context.Context, customerID, referenceID uuid.UUID, referenceType tstranscribe.Type, language string) (*tstranscribe.Transcribe, error) {
	uri := "/v1/streamings"

	data := &tmrequest.V1DataStreamingsPost{
		CustomerID:    customerID,
		ReferenceID:   referenceID,
		ReferenceType: referenceType,
		Language:      language,
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

	var res tstranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
