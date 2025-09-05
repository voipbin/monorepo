package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	carequest "monorepo/bin-campaign-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// CampaignV1CampaignCreate creates a new campaign.
func (r *requestHandler) CampaignV1CampaignCreate(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	campaignType cacampaign.Type,
	name string,
	detail string,
	serviceLevel int,
	endHandle cacampaign.EndHandle,
	actions []fmaction.Action,
	outplanID uuid.UUID,
	outdialID uuid.UUID,
	queueID uuid.UUID,
	nextCampaignID uuid.UUID,
) (*cacampaign.Campaign, error) {

	uri := "/v1/campaigns"

	reqData := &carequest.V1DataCampaignsPost{
		ID:             id,
		CustomerID:     customerID,
		Type:           campaignType,
		Name:           name,
		Detail:         detail,
		ServiceLevel:   serviceLevel,
		EndHandle:      endHandle,
		Actions:        actions,
		OutplanID:      outplanID,
		OutdialID:      outdialID,
		QueueID:        queueID,
		NextCampaignID: nextCampaignID,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPost, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignGetsByCustomerID sends a request to campaign-manager
// to getting a list of campaigns.
// it returns detail list of campaigns if it succeed.
func (r *requestHandler) CampaignV1CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CampaignV1CampaignGet sends a request to campaign-manager
// to getting a detail campaign info.
// it returns detail campaign info if it succeed.
func (r *requestHandler) CampaignV1CampaignGet(ctx context.Context, id uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodGet, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignDelete sends a request to campaign-manager
// to deleting the campaign.
func (r *requestHandler) CampaignV1CampaignDelete(ctx context.Context, campaignID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", campaignID)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodDelete, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignExecute sends a request to campaign-manager
// to execute the campaign.
// delay millisecond
func (r *requestHandler) CampaignV1CampaignExecute(ctx context.Context, id uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/campaigns/%s/execute", id)

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPost, "campaign/campaigns", requestTimeoutDefault, delay, ContentTypeJSON, nil)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}

// CampaignV1CampaignUpdateBasicInfo sends a request to campaign-manager
// to update the campaign's basic info.
// delay millisecond
func (r *requestHandler) CampaignV1CampaignUpdateBasicInfo(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	campaignType cacampaign.Type,
	serviceLevel int,
	endHandle cacampaign.EndHandle,
) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", id)

	data := &carequest.V1DataCampaignsIDPut{
		Name:         name,
		Detail:       detail,
		Type:         campaignType,
		ServiceLevel: serviceLevel,
		EndHandle:    endHandle,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignUpdateStatus sends a request to campaign-manager
// to update the status.
// it returns updated campaign if it succeed.
func (r *requestHandler) CampaignV1CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status cacampaign.Status) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/status", id)

	data := &carequest.V1DataCampaignsIDStatusPut{
		Status: status,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignUpdateServiceLevel sends a request to campaign-manager
// to update the service_level.
// it returns updated campaign if it succeed.
func (r *requestHandler) CampaignV1CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/service_level", id)

	data := &carequest.V1DataCampaignsIDServiceLevelPut{
		ServiceLevel: serviceLevel,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignUpdateActions sends a request to campaign-manager
// to update the actions.
// it returns updated campaign if it succeed.
func (r *requestHandler) CampaignV1CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/actions", id)

	data := &carequest.V1DataCampaignsIDActionsPut{
		Actions: actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignUpdateResourceInfo sends a request to campaign-manager
// to update the resource_info.
// it returns updated campaign if it succeed.
func (r *requestHandler) CampaignV1CampaignUpdateResourceInfo(ctx context.Context, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/resource_info", id)

	data := &carequest.V1DataCampaignsIDResourceInfoPut{
		OutplanID:      outplanID,
		OutdialID:      outdialID,
		QueueID:        queueID,
		NextCampaignID: nextCampaignID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CampaignV1CampaignUpdateNextCampaignID sends a request to campaign-manager
// to update the next_campaign_id.
// it returns updated campaign if it succeed.
func (r *requestHandler) CampaignV1CampaignUpdateNextCampaignID(ctx context.Context, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/next_campaign_id", id)

	data := &carequest.V1DataCampaignsIDNextCampaignIDPut{
		NextCampaignID: nextCampaignID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(ctx, uri, sock.RequestMethodPut, "campaign/campaigns", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cacampaign.Campaign
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
