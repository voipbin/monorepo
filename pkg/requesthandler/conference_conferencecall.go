package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	cfrequest "gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ConferenceV1ConferencecallGet gets the conferencecall.
func (r *requestHandler) ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {

	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceConferenceConferencecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// ConferenceV1ConferencecallCreate sends a request to conference-manager
// to creating a conferencecall.
// it returns created conference if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallCreate(ctx context.Context, conferenceID uuid.UUID, referenceType cfconferencecall.ReferenceType, referenceID uuid.UUID) (*cfconferencecall.Conferencecall, error) {
	uri := "/v1/conferencecalls"

	d := &cfrequest.V1DataConferencecallsPost{
		ConferenceID:  conferenceID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	}

	m, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceConferenceConferences, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
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

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceConferenceConferencecalls, requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

// ConferenceV1ConferencecallHealthCheck sends a request to conference-manager
// to checks the health of the given conferencecall.
// it returns kicked conferencecall if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallHealthCheck(ctx context.Context, conferencecallID uuid.UUID, retryCount int) (*cfconferencecall.Conferencecall, error) {
	uri := fmt.Sprintf("/v1/conferencecalls/%s/health-check", conferencecallID)

	d := &cfrequest.V1DataConferencecallsIDHealthCheckPost{
		RetryCount: retryCount,
	}

	m, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceConferenceConferencecalls, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
