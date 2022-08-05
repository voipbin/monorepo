package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ConferenceV1ConferencecallGet gets the conferencecall.
func (r *requestHandler) ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {

	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodGet, resourceCFConferencecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res cfconferencecall.Conferencecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConferenceV1ConferencecallKick sends a request to conference-manager
// to kick the given conferencecall from the conference
// it returns kicked conferencecall if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallKick(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {
	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(uri, rabbitmqhandler.RequestMethodDelete, resourceCFConferencecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cfconferencecall.Conferencecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
