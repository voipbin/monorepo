package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	cmconference "gitlab.com/voipbin/bin-manager/call-manager.git/models/conference"
	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CMConferenceGets sends a request to call-manager
// to getting a list of conference info.
// it returns detail list of conference info if it succeed.
func (r *requestHandler) CMConferenceGets(userID uint64, pageToken string, pageSize uint64) ([]cmconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodGet, resourceCallCall, 30, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var confs []cmconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &confs); err != nil {
		return nil, err
	}

	return confs, nil
}

// CMConferenceGet sends a request to call-manager
// to getting a conference information.
// it returns created conference if it succeed.
func (r *requestHandler) CMConferenceGet(conferenceID uuid.UUID) (*cmconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodGet, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference cmconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}

// CMConferenceDelete sends a request to call-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) CMConferenceDelete(conferenceID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodDelete, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMConferenceCreate sends a request to call-manager
// to creating a conference.
// it returns created conference if it succeed.
func (r *requestHandler) CMConferenceCreate(userID uint64, conferenceType cmconference.Type, name string, detail string, webhookURI string) (*cmconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences")

	data := &cmrequest.V1DataConferencesIDPost{
		Type:       conferenceType,
		UserID:     userID,
		Name:       name,
		Detail:     detail,
		Timeout:    86400, // 24 hour
		WebhookURI: webhookURI,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(uri, rabbitmqhandler.RequestMethodPost, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference cmconference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}
