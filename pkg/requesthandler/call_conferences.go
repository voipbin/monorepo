package requesthandler

import (
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/rabbitmq/models"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/conference"
)

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
