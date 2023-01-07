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

// CallV1RecordingGets sends a request to call-manager
// to getting recordings.
// it returns list of recordings if it succeed.
func (r *requestHandler) CallV1RecordingGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(token), size, customerID)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1RecordingGet sends a request to call-manager
// to getting a recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1RecordingDelete sends a request to call-manager
// to delete a recording.
// it returns deleted recording if it succeed.
func (r *requestHandler) CallV1RecordingDelete(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	tmp, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceCallRecordings, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cmrecording.Recording
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
