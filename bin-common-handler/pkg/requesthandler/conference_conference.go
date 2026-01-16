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
	"github.com/pkg/errors"
)

// ConferenceV1ConferenceGet gets the conference.
func (r *requestHandler) ConferenceV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {

	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferences", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConferenceV1ConferenceGets sends a request to conference-manager
// to getting a list of conference info.
// it returns detail list of conference info if it succeed.
func (r *requestHandler) ConferenceV1ConferenceList(ctx context.Context, pageToken string, pageSize uint64, filters map[cfconference.Field]any) ([]cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferences", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ConferenceV1ConferenceDelete sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodDelete, "conference/conferences/<conference-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConferenceV1ConferenceDeleteDelay sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) ConferenceV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodDelete, "conference/conferences", requestTimeoutDefault, delay, ContentTypeNone, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConferenceV1ConferenceRecordingStop sends a request to conference-manager
// to stop the conference recording.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceRecordingStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/recording_stop", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/recording_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConferenceV1ConferenceTranscribeStop sends a request to conference-manager
// to stop the conference transcribe.
// it returns error if it failed.
func (r *requestHandler) ConferenceV1ConferenceTranscribeStop(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s/transcribe_stop", conferenceID.String())

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferences/<conference-id>/transcribe_stop", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cfconference.Conference
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
