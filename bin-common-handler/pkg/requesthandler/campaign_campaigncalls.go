package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// CampaignV1CampaigncallGets sends a request to campaign-manager
// to getting a list of campaigncalls.
// it returns detail list of campaigncalls if it succeed.
func (r *requestHandler) CampaignV1CampaigncallGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cacampaigncall.Campaigncall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CampaignV1CampaigncallGetsByCampaignID sends a request to campaign-manager
// to getting a list of campaigncalls of the given campaign id.
// it returns detail list of campaigncalls if it succeed.
func (r *requestHandler) CampaignV1CampaigncallGetsByCampaignID(ctx context.Context, campaignID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls?page_token=%s&page_size=%d&campaign_id=%s", url.QueryEscape(pageToken), pageSize, campaignID)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cacampaigncall.Campaigncall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CampaignV1CampaigncallGet sends a request to campaign-manager
// to getting a detail campaigncall.
// it returns detail campaigncall if it succeed.
func (r *requestHandler) CampaignV1CampaigncallGet(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaigncall.Campaigncall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CampaignV1CampaigncallDelete sends a request to campaign-manager
// to delete(hangup) the campaigncall.
// it returns detail campaigncall if it succeed.
func (r *requestHandler) CampaignV1CampaigncallDelete(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodDelete, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaigncall.Campaigncall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
