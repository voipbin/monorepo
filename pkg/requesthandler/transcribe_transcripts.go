package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// TranscribeV1TranscriptGets sends a request to transcribe-manager
// to getting a list of transcript info.
// it returns detail list of transcript info if it succeed.
func (r *requestHandler) TranscribeV1TranscriptGets(ctx context.Context, transcribeID uuid.UUID) ([]tmtranscript.Transcript, error) {
	uri := fmt.Sprintf("/v1/transcripts?transcribe_id=%s", transcribeID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceTranscribeTranscripts, 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []tmtranscript.Transcript
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
