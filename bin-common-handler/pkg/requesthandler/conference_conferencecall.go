package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	cfrequest "monorepo/bin-conference-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// ConferenceV1ConferencecallGets sends a request to conference-manager
// to getting a list of conferencecalls info.
// it returns detail list of conferencecalls info if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cfconferencecall.Conferencecall, error) {
	uri := fmt.Sprintf("/v1/conferencecalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodGet, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cfconferencecall.Conferencecall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ConferenceV1ConferencecallGet gets the conferencecall.
func (r *requestHandler) ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {

	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodGet, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodDelete, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
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
// delay: milliseconds
func (r *requestHandler) ConferenceV1ConferencecallHealthCheck(ctx context.Context, conferencecallID uuid.UUID, retryCount int, delay int) error {
	uri := fmt.Sprintf("/v1/conferencecalls/%s/health-check", conferencecallID)

	d := &cfrequest.V1DataConferencecallsIDHealthCheckPost{
		RetryCount: retryCount,
	}

	m, err := json.Marshal(d)
	if err != nil {
		return err
	}

	tmp, err := r.sendRequestConference(ctx, uri, rabbitmqhandler.RequestMethodPost, "conference/conferencecalls", requestTimeoutDefault, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		return nil
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}
