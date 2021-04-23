package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	tmrequest "gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/listenhandler/models/request"
)

// TMRecordingPost sends a request to transcode-manager
// to transcode the exist recording.
// it returns transcoded text if it succeed.
func (r *requestHandler) TMRecordingPost(id uuid.UUID, language string) (*tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/recordings")

	req := &tmrequest.V1DataRecordingsPost{
		ReferenceID: id,
		Language:    language,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestTranscribe(uri, rabbitmqhandler.RequestMethodPost, resourceStorageRecording, 60, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var data tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(res.Data), &data); err != nil {
		return nil, err
	}

	return &data, nil
}
