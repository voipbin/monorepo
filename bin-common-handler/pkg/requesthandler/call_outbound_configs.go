package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// CallV1OutboundConfigCreate sends a request to call-manager to create an outbound config.
func (r *requestHandler) CallV1OutboundConfigCreate(ctx context.Context, customerID uuid.UUID, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.OutboundConfig, error) {
	uri := "/v1/outbound_configs"

	body := struct {
		CustomerID uuid.UUID                    `json:"customer_id"`
		Request    *cmoutboundconfig.UpdateRequest `json:"request"`
	}{
		CustomerID: customerID,
		Request:    req,
	}
	m, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/outbound_configs", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmoutboundconfig.OutboundConfig
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1OutboundConfigGet sends a request to call-manager to get an outbound config by ID.
func (r *requestHandler) CallV1OutboundConfigGet(ctx context.Context, id uuid.UUID) (*cmoutboundconfig.OutboundConfig, error) {
	uri := fmt.Sprintf("/v1/outbound_configs/%s", id)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/outbound_configs", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cmoutboundconfig.OutboundConfig
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1OutboundConfigList sends a request to call-manager to list outbound configs.
func (r *requestHandler) CallV1OutboundConfigList(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]cmoutboundconfig.OutboundConfig, error) {
	uri := fmt.Sprintf("/v1/outbound_configs?customer_id=%s&page_size=%d&page_token=%s", customerID, pageSize, url.QueryEscape(pageToken))

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/outbound_configs", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []cmoutboundconfig.OutboundConfig
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CallV1OutboundConfigDelete sends a request to call-manager to delete an outbound config.
func (r *requestHandler) CallV1OutboundConfigDelete(ctx context.Context, id uuid.UUID) (*cmoutboundconfig.OutboundConfig, error) {
	uri := fmt.Sprintf("/v1/outbound_configs/%s", id)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodDelete, "call/outbound_configs", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cmoutboundconfig.OutboundConfig
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CallV1OutboundConfigUpdate sends a request to call-manager to update an outbound config.
func (r *requestHandler) CallV1OutboundConfigUpdate(ctx context.Context, id uuid.UUID, req *cmoutboundconfig.UpdateRequest) (*cmoutboundconfig.OutboundConfig, error) {
	uri := fmt.Sprintf("/v1/outbound_configs/%s", id)

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPut, "call/outbound_configs", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cmoutboundconfig.OutboundConfig
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
