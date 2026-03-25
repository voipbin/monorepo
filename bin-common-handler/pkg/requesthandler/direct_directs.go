package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// DirectV1DirectCreate sends a request to direct-manager
// to creating a direct.
// it returns created direct if it succeed.
func (r *requestHandler) DirectV1DirectCreate(ctx context.Context, customerID uuid.UUID, resourceType string, resourceID uuid.UUID) (*dmdirect.Direct, error) {
	uri := "/v1/directs"

	type reqBody struct {
		CustomerID   uuid.UUID `json:"customer_id"`
		ResourceType string    `json:"resource_type"`
		ResourceID   uuid.UUID `json:"resource_id"`
	}

	data := &reqBody{
		CustomerID:   customerID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodPost, "direct/directs", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// DirectV1DirectGet sends a request to direct-manager
// to getting the direct.
func (r *requestHandler) DirectV1DirectGet(ctx context.Context, id uuid.UUID) (*dmdirect.Direct, error) {
	uri := fmt.Sprintf("/v1/directs/%s", id)

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodGet, "direct/directs", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// DirectV1DirectGetByHash sends a request to direct-manager
// to getting the direct by hash.
func (r *requestHandler) DirectV1DirectGetByHash(ctx context.Context, hash string) (*dmdirect.Direct, error) {
	uri := fmt.Sprintf("/v1/directs/by-hash/%s", url.PathEscape(hash))

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodGet, "direct/directs", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// DirectV1DirectGets sends a request to direct-manager
// to getting the list of directs.
func (r *requestHandler) DirectV1DirectGets(ctx context.Context, pageToken string, pageSize uint64, filters map[dmdirect.Field]any) ([]*dmdirect.Direct, error) {
	uri := fmt.Sprintf("/v1/directs?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodGet, "direct/directs", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// DirectV1DirectDelete sends a request to direct-manager
// to deleting the direct.
func (r *requestHandler) DirectV1DirectDelete(ctx context.Context, id uuid.UUID) (*dmdirect.Direct, error) {
	uri := fmt.Sprintf("/v1/directs/%s", id)

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodDelete, "direct/directs", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// DirectV1DirectRegenerate sends a request to direct-manager
// to regenerate the direct hash.
func (r *requestHandler) DirectV1DirectRegenerate(ctx context.Context, id uuid.UUID) (*dmdirect.Direct, error) {
	uri := fmt.Sprintf("/v1/directs/%s/regenerate", id)

	tmp, err := r.sendRequestDirect(ctx, uri, sock.RequestMethodPost, "direct/directs", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res dmdirect.Direct
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
