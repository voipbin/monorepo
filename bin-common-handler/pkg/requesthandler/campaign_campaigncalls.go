package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// CampaignV1CampaigncallList sends a request to campaign-manager
// to getting a list of campaigncalls.
// it returns detail list of campaigncalls if it succeed.
func (r *requestHandler) CampaignV1CampaigncallList(ctx context.Context, pageToken string, pageSize uint64, filters map[cacampaigncall.Field]any) ([]cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cacampaigncall.Campaigncall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CampaignV1CampaigncallGet sends a request to campaign-manager
// to getting a detail campaigncall.
// it returns detail campaigncall if it succeed.
func (r *requestHandler) CampaignV1CampaigncallGet(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cacampaigncall.Campaigncall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaigncallDelete sends a request to campaign-manager
// to delete(hangup) the campaigncall.
// it returns detail campaigncall if it succeed.
func (r *requestHandler) CampaignV1CampaigncallDelete(ctx context.Context, id uuid.UUID) (*cacampaigncall.Campaigncall, error) {
	uri := fmt.Sprintf("/v1/campaigncalls/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodDelete, "campaign/campaigncalls", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cacampaigncall.Campaigncall
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
