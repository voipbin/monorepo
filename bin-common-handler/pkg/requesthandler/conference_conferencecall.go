package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	cfrequest "monorepo/bin-conference-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ConferenceV1ConferencecallGets sends a request to conference-manager
// to getting a list of conferencecalls info.
// it returns detail list of conferencecalls info if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallList(ctx context.Context, pageToken string, pageSize uint64, filters map[cfconferencecall.Field]any) ([]cfconferencecall.Conferencecall, error) {
	uri := fmt.Sprintf("/v1/conferencecalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cfconferencecall.Conferencecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ConferenceV1ConferencecallGet gets the conferencecall.
func (r *requestHandler) ConferenceV1ConferencecallGet(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {

	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodGet, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cfconferencecall.Conferencecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConferenceV1ConferencecallKick sends a request to conference-manager
// to kick the given conferencecall from the conference
// it returns kicked conferencecall if it succeed.
func (r *requestHandler) ConferenceV1ConferencecallKick(ctx context.Context, conferencecallID uuid.UUID) (*cfconferencecall.Conferencecall, error) {
	uri := fmt.Sprintf("/v1/conferencecalls/%s", conferencecallID)

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodDelete, "conference/conferencecalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cfconferencecall.Conferencecall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/conferencecalls", requestTimeoutDefault, delay, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
