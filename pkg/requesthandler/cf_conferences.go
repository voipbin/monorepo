package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	cfrequest "gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"
)

// CMConferenceGets sends a request to call-manager
// to getting a list of conference info.
// it returns detail list of conference info if it succeed.
func (r *requestHandler) CFConferenceGets(userID uint64, pageToken string, pageSize uint64, conferenceType string) ([]cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences?page_token=%s&page_size=%d&user_id=%d&type=%s", url.QueryEscape(pageToken), pageSize, userID, conferenceType)

	res, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodGet, resourceConferenceConference, 30, 0, ContentTypeJSON, nil)
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

// CMConferenceGet sends a request to call-manager
// to getting a conference information.
// it returns created conference if it succeed.
func (r *requestHandler) CFConferenceGet(conferenceID uuid.UUID) (*cfconference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodGet, resourceConferenceConference, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// CMConferenceDelete sends a request to call-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) CFConferenceDelete(conferenceID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodDelete, resourceConferenceConference, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
func (r *requestHandler) CFConferenceCreate(userID uint64, conferenceType cfconference.Type, name string, detail string, webhookURI string) (*cfconference.Conference, error) {
	uri := "/v1/conferences"

	data := &cfrequest.V1DataConferencesPost{
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

	res, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodPost, resourceConferenceConference, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
