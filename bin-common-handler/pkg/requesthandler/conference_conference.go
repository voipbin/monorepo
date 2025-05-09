package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmrecording "monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/models/sock"
	cfconference "monorepo/bin-conference-manager/models/conference"
	cfrequest "monorepo/bin-conference-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// ConferenceV1ConferenceGet gets the conference.
func (r *requestHandler) ConferenceV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {

	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferences", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceGets sends a request to conference-manager
// to getting a list of conference info.
// it returns detail list of conference info if it succeed.
func (r *requestHandler) ConferenceV1ConferenceGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferences", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ConferenceV1ConferenceDelete sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodDelete, "conference/conferences/<conference-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceDeleteDelay sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestConference(ctx, uri, sock.RequestMethodDelete, "conference/conferences", requestTimeoutDefault, delay, ContentTypeNone, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// ConferenceV1ConferenceStop sends a request to conference-manager
// to stop the conference.
// it returns deleted conference if it succeed.
// delay: milliseconds
func (r *requestHandler) ConferenceV1ConferenceStop(ctx context.Context, conferenceID uuid.UUID, delay int) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/stop", conferenceID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>", requestTimeoutDefault, delay, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, nil
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceCreate sends a request to conference-manager
// to creating a conference.
// it returns created conference if it succeed.
// timeout(sec)
// it the timeout set to 0 means no timeout.
func (r *requestHandler) ConferenceV1ConferenceCreate(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	conferenceType cfconference.Type,
	name string,
	detail string,
	data map[string]interface{},
	timeout int,
	preFlowID uuid.UUID,
	postFlowID uuid.UUID,
) (*cfconference.Conference, error) {
	uri := "/v1/conferences"

	d := &cfrequest.V1DataConferencesPost{
		ID:         id,
		CustomerID: customerID,
		Type:       conferenceType,
		Name:       name,
		Detail:     detail,
		Data:       data,
		Timeout:    timeout,
		PreFlowID:  preFlowID,
		PostFlowID: postFlowID,
	}

	m, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceUpdate sends a request to conference-manager
// to update the conference.
// it returns updated conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	data map[string]any,
	timeout int,
	preFlowID uuid.UUID,
	postFlowID uuid.UUID,
) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", id.String())

	reqData := &cfrequest.V1DataConferencesIDPut{
		Name:       name,
		Detail:     detail,
		Data:       data,
		Timeout:    timeout,
		PreFlowID:  preFlowID,
		PostFlowID: postFlowID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPut, "conference/conferences", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceUpdateRecordingID sends a request to conference-manager
// to update the conference's recording id.
// it returns updated conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceUpdateRecordingID(ctx context.Context, id uuid.UUID, recordingID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/recording_id", id.String())

	data := &cfrequest.V1DataConferencesIDRecordingIDPut{
		RecordingID: recordingID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPut, "conference/conferences", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceRecordingStart sends a request to conference-manager
// to start the conference recording.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceRecordingStart(
	ctx context.Context,
	conferenceID uuid.UUID,
	activeflowID uuid.UUID,
	format cmrecording.Format,
	duration int,
	onEndFlowID uuid.UUID,
) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/recording_start", conferenceID.String())

	data := &cfrequest.V1DataConferencesIDRecordingStartPost{
		ActiveflowID: activeflowID,
		Format:       format,
		Duration:     duration,
		OnEndFlowID:  onEndFlowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/recording_start", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceRecordingStop sends a request to conference-manager
// to stop the conference recording.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceRecordingStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/recording_stop", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/recording_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceTranscribeStart sends a request to conference-manager
// to start the conference transcribe.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceTranscribeStart(ctx context.Context, conferenceID uuid.UUID, language string) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/transcribe_start", conferenceID.String())

	data := &cfrequest.V1DataConferencesIDTranscribeStartPost{
		Language: language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/transdribe_start", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferenceTranscribeStop sends a request to conference-manager
// to stop the conference transcribe.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceTranscribeStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/transcribe_stop", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/transcribe_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconference.Conference
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
