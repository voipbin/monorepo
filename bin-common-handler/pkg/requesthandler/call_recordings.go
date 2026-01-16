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
	"github.com/pkg/errors"
)

// CallV1RecordingList sends a request to call-manager
// to getting recordings.
// it returns list of recordings if it succeed.
func (r *requestHandler) CallV1RecordingList(ctx context.Context, pageToken string, pageSize uint64, filters map[cmrecording.Field]any) ([]cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cmrecording.Recording
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CallV1RecordingGet sends a request to call-manager
// to getting a recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingGet(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cmrecording.Recording
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1RecordingDelete sends a request to call-manager
// to delete a recording.
// it returns deleted recording if it succeed.
func (r *requestHandler) CallV1RecordingDelete(ctx context.Context, id uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s", id)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/recordings", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cmrecording.Recording
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1RecordingStart sends a request to call-manager
// to start a recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingStart(
	ctx context.Context,
	activeflowID uuid.UUID,
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
		ActiveflowID:  activeflowID,
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
	if err != nil {
		return nil, err
	}

	var res cmrecording.Recording
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1RecordingStop sends a request to call-manager
// to stop the recording.
// it returns recording if it succeed.
func (r *requestHandler) CallV1RecordingStop(ctx context.Context, recordingID uuid.UUID) (*cmrecording.Recording, error) {
	uri := fmt.Sprintf("/v1/recordings/%s/stop", recordingID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/recordings", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cmrecording.Recording
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
