package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/rabbitmq/models"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/conference"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/request"
)

// CallConferenceGet sends a request to call-manager
// to getting a conference information.
// it returns created conference if it succeed.
func (r *requestHandler) CallConferenceGet(conferenceID uuid.UUID) (*conference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCall(uri, models.RequestMethodGet, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, "")
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference conference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}

// CallConferenceDelete sends a request to call-manager
// to deleting a conference.
// it returns deleted conference if it succeed.
func (r *requestHandler) CallConferenceDelete(conferenceID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/conferences/%s", conferenceID)

	res, err := r.sendRequestCall(uri, models.RequestMethodDelete, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, "")
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

// CallConferenceCreate sends a request to call-manager
// to creating a conference.
// it returns created conference if it succeed.
func (r *requestHandler) CallConferenceCreate(conferenceType conference.Type) (*conference.Conference, error) {
	uri := fmt.Sprintf("/v1/conferences")

	data := &request.V1DataConferencesCreate{
		Type: conferenceType,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCall(uri, models.RequestMethodPost, resourceCallConference, requestTimeoutDefault, 0, ContentTypeJSON, string(m))
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var conference conference.Conference
	if err := json.Unmarshal([]byte(res.Data), &conference); err != nil {
		return nil, err
	}

	return &conference, nil
}
