package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	caoutplan "monorepo/bin-campaign-manager/models/outplan"
	carequest "monorepo/bin-campaign-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
)

// CampaignV1OutplanCreate creates a new outplan.
func (r *requestHandler) CampaignV1OutplanCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	source *address.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.Outplan, error) {

	uri := "/v1/outplans"

	reqData := &carequest.V1DataOutplansPost{
		CustomerID:   customerID,
		Name:         name,
		Detail:       detail,
		Source:       source,
		DialTimeout:  dialTimeout,
		TryInterval:  tryInterval,
		MaxTryCount0: maxTryCount0,
		MaxTryCount1: maxTryCount1,
		MaxTryCount2: maxTryCount2,
		MaxTryCount3: maxTryCount3,
		MaxTryCount4: maxTryCount4,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPost, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1OutplanGets sends a request to campaign-manager
// to getting a list of outplans.
// it returns detail list of outplans if it succeed.
func (r *requestHandler) CampaignV1OutplanGets(ctx context.Context, pageToken string, pageSize uint64, filters map[caoutplan.Field]any) ([]caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CampaignV1OutplanGet sends a request to campaign-manager
// to getting a detail outplan info.
// it returns detail outplan info if it succeed.
func (r *requestHandler) CampaignV1OutplanGet(ctx context.Context, id uuid.UUID) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1OutplanDelete sends a request to campaign-manager
// to deleting the outplan.
func (r *requestHandler) CampaignV1OutplanDelete(ctx context.Context, outplanID uuid.UUID) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", outplanID)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodDelete, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1OutplanUpdateBasicInfo sends a request to campaign-manager
// to update the outplan's basic info.
// delay millisecond
func (r *requestHandler) CampaignV1OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", id)

	data := &carequest.V1DataOutplansIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1OutplanUpdateDialInfo sends a request to campaign-manager
// to update the outplan's dial info.
// delay millisecond
func (r *requestHandler) CampaignV1OutplanUpdateDialInfo(
	ctx context.Context,
	id uuid.UUID,
	source *address.Address,
	dialTimeout int,
	tryInterval int,
	maxTryCount0 int,
	maxTryCount1 int,
	maxTryCount2 int,
	maxTryCount3 int,
	maxTryCount4 int,
) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s/dials", id)

	data := &carequest.V1DataOutplansIDDialsPut{
		Source:       source,
		DialTimeout:  dialTimeout,
		TryInterval:  tryInterval,
		MaxTryCount0: maxTryCount0,
		MaxTryCount1: maxTryCount1,
		MaxTryCount2: maxTryCount2,
		MaxTryCount3: maxTryCount3,
		MaxTryCount4: maxTryCount4,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/outplans", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res caoutplan.Outplan
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
