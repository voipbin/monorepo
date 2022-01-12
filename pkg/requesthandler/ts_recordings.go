package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tstranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tsrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// TSV1RecordingCreate sends a request to transcribe-manager
// to transcode the exist recording.
// it returns transcoded text if it succeed.
func (r *requestHandler) TSV1RecordingCreate(ctx context.Context, id uuid.UUID, language string) (*tstranscribe.Transcribe, error) {
	uri := "/v1/recordings"

	req := &tsrequest.V1DataRecordingsPost{
		ReferenceID: id,
		Language:    language,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestTS(uri, rabbitmqhandler.RequestMethodPost, resourceStorageRecording, 60, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data tstranscribe.Transcribe
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return &data, nil
}
