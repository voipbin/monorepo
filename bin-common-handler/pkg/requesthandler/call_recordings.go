package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmrecording "monorepo/bin-call-manager/models/recording"
	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// CallV1RecordingGets sends a request to call-manager
// to getting recordings.
// it returns list of recordings if it succeed.
func (r *requestHandler) CallV1RecordingGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	res, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

	res, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CallV1RecordingStart sends a request to call-manager
// to start a recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingStart(
	ctx context.Context,
	referenceType cmrecording.ReferenceType,
	referenceID uuid.UUID,
	format cmrecording.Format,
	endOfSilence int,
	endOfKey string,
	duration int,
	onEndFlowID uuid.UUID,
) (*cmrecording.Recording, error) {
	uri := "/v1/recordings"

	reqData := &cmrequest.V1DataRecordingsPost{
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Format:        format,
		EndOfSilence:  endOfSilence,
		EndOfKey:      endOfKey,
		Duration:      duration,
		OnEndFlowID:   onEndFlowID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// CallV1RecordingStop sends a request to call-manager
// to stop the recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingStop(ctx context.Context, recordingID uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s/stop", recordingID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/recordings", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
