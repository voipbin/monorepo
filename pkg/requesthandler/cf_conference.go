package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cfrequest "gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CFV1ConferenceGet gets the conference.
func (r *requestHandler) CFV1ConferenceGet(ctx context.Context, conferenceID uuid.UUID) (*cfconference.Conference, error) {

	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID.String())

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodGet, resourceCFConferences, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var resData cfconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CFV1ConferenceGets sends a request to conference-manager
// to getting a list of conference info.
// it returns detail list of conference info if it succeed.
func (r *requestHandler) CFV1ConferenceGets(ctx context.Context, userID uint64, pageToken string, pageSize uint64, conferenceType string) ([]cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences?page_token=%s&page_size=%d&user_id=%d&type=%s", url.QueryEscape(pageToken), pageSize, userID, conferenceType)

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodGet, resourceCFConferences, 30000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var confs []cfconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &confs); err != nil {
		return nil, err
	}

	return confs, nil
}

// CFV1ConferenceDelete sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) CFV1ConferenceDelete(ctx context.Context, conferenceID uuid.UUID) error {
	return r.CFV1ConferenceDeleteDelay(ctx, conferenceID, DelayNow)
}

// CFV1ConferenceDeleteDelay sends a request to conference-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) CFV1ConferenceDeleteDelay(ctx context.Context, conferenceID uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodDelete, resourceCFConferences, requestTimeoutDefault, delay, ContentTypeJSON, []byte(""))
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

// CFV1ConferenceCreate sends a request to conference-manager
// to creating a conference.
// it returns created conference if it succeed.
// timeout(sec)
// it the timeout set to 0 means no timeout.
func (r *requestHandler) CFV1ConferenceCreate(
	ctx context.Context,
	userID uint64,
	conferenceType cfconference.Type,
	name string,
	detail string,
	timeout int,
	webhookURI string,
	data map[string]interface{},
	preActions []fmaction.Action,
	postActions []fmaction.Action,
) (*cfconference.Conference, error) {
	uri := "/v1/conferences"

	d := &cfrequest.V1DataConferencesPost{
		Type:        conferenceType,
		UserID:      userID,
		Name:        name,
		Detail:      detail,
		Timeout:     timeout,
		WebhookURI:  webhookURI,
		Data:        data,
		PreActions:  preActions,
		PostActions: postActions,
	}

	m, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodPost, resourceCFConferences, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference cfconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}

// CFV1ConferenceUpdate sends a request to conference-manager
// to update the conference.
// it returns updated conference if it succeed.
func (r *requestHandler) CFV1ConferenceUpdate(ctx context.Context, id uuid.UUID, name string, detail string, timeout int, webhookURI string, preActions, postActions []fmaction.Action) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", id.String())

	data := &cfrequest.V1DataConferencesIDPut{
		Name:        name,
		Detail:      detail,
		Timeout:     timeout,
		WebhookURI:  webhookURI,
		PreActions:  preActions,
		PostActions: postActions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodPut, resourceCFConferences, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference cfconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}

// CFV1ConferenceKick sends a request to conference-manager
// to kick the call from the conference
// it returns deleted conference if it succeed.
func (r *requestHandler) CFV1ConferenceKick(ctx context.Context, conferenceID, callID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/conferences/%s/calls/%s", conferenceID, callID)

	res, err := r.sendRequestCF(uri, rabbitmqhandler.RequestMethodDelete, resourceCFConferences, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
