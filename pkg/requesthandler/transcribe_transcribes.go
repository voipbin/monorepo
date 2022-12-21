package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// TranscribeV1TranscribeGet sends a request to transcribe-manager
// to getting a call.
// it returns given transcribe id's transcribe if it succeed.
func (r *requestHandler) TranscribeV1TranscribeGet(ctx context.Context, transcribeID uuid.UUID) (*tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes/%s", transcribeID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// TranscribeV1TranscribeGets sends a request to transcribe-manager
// to getting a list of transcribe info.
// it returns detail list of transcribe info if it succeed.
func (r *requestHandler) TranscribeV1TranscribeGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]tmtranscribe.Transcribe, error) {
	uri := fmt.Sprintf("/v1/transcribes?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestTranscribe(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallCalls, 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []tmtranscribe.Transcribe
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
