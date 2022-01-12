package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMV1RecordingGets sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMV1RecordingGets(ctx context.Context, userID uint64, size uint64, token string) ([]cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(token), size, userID)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodGet, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var recordings []cmrecording.Recording
	if err := json.Unmarshal([]byte(res.Data), &recordings); err != nil {
		return nil, err
	}

	return recordings, nil
}

// CMV1RecordingGet sends a request to call-manager
// to creating a call.
// it returns created call if it succeed.
func (r *requestHandler) CMV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	res, err := r.sendRequestCM(uri, rabbitmqhandler.RequestMethodGet, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var c cmrecording.Recording
	if err := json.Unmarshal([]byte(res.Data), &c); err != nil {
		return nil, err
	}

	return &c, nil
}
