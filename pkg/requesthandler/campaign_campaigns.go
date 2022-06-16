package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	carequest "gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/listenhandler/models/request"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CAV1CampaignCreate creates a new campaign.
func (r *requestHandler) CAV1CampaignCreate(
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

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPost, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not create a campaign. status: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignGetsByCustomerID sends a request to campaign-manager
// to getting a list of campaigns.
// it returns detail list of campaigns if it succeed.
func (r *requestHandler) CAV1CampaignGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodGet, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CAV1CampaignGet sends a request to campaign-manager
// to getting a detail campaign info.
// it returns detail campaign info if it succeed.
func (r *requestHandler) CAV1CampaignGet(ctx context.Context, id uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", id)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodGet, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignDelete sends a request to campaign-manager
// to deleting the campaign.
func (r *requestHandler) CAV1CampaignDelete(ctx context.Context, campaignID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", campaignID)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodDelete, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignExecute sends a request to campaign-manager
// to execute the campaign.
// delay millisecond
func (r *requestHandler) CAV1CampaignExecute(ctx context.Context, id uuid.UUID, delay int) error {
	uri := fmt.Sprintf("/v1/campaigns/%s/execute", id)

	res, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPost, resourceCACampaigns, requestTimeoutDefault, delay, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}

// CAV1CampaignUpdateBasicInfo sends a request to campaign-manager
// to update the campaign's basic info.
// delay millisecond
func (r *requestHandler) CAV1CampaignUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s", id)

	data := &carequest.V1DataCampaignsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignUpdateStatus sends a request to campaign-manager
// to update the status.
// it returns updated campaign if it succeed.
func (r *requestHandler) CAV1CampaignUpdateStatus(ctx context.Context, id uuid.UUID, status cacampaign.Status) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/status", id)

	data := &carequest.V1DataCampaignsIDStatusPut{
		Status: status,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignUpdateServiceLevel sends a request to campaign-manager
// to update the service_level.
// it returns updated campaign if it succeed.
func (r *requestHandler) CAV1CampaignUpdateServiceLevel(ctx context.Context, id uuid.UUID, serviceLevel int) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/service_level", id)

	data := &carequest.V1DataCampaignsIDServiceLevelPut{
		ServiceLevel: serviceLevel,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignUpdateActions sends a request to campaign-manager
// to update the actions.
// it returns updated campaign if it succeed.
func (r *requestHandler) CAV1CampaignUpdateActions(ctx context.Context, id uuid.UUID, actions []fmaction.Action) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/actions", id)

	data := &carequest.V1DataCampaignsIDActionsPut{
		Actions: actions,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignUpdateResourceInfo sends a request to campaign-manager
// to update the resource_info.
// it returns updated campaign if it succeed.
func (r *requestHandler) CAV1CampaignUpdateResourceInfo(ctx context.Context, id uuid.UUID, outplanID uuid.UUID, outdialID uuid.UUID, queueID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/resource_info", id)

	data := &carequest.V1DataCampaignsIDResourceInfoPut{
		OutplanID: outplanID,
		OutdialID: outdialID,
		QueueID:   queueID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1CampaignUpdateNextCampaignID sends a request to campaign-manager
// to update the next_campaign_id.
// it returns updated campaign if it succeed.
func (r *requestHandler) CAV1CampaignUpdateNextCampaignID(ctx context.Context, id uuid.UUID, nextCampaignID uuid.UUID) (*cacampaign.Campaign, error) {
	uri := fmt.Sprintf("/v1/campaigns/%s/next_campaign_id", id)

	data := &carequest.V1DataCampaignsIDNextCampaignIDPut{
		NextCampaignID: nextCampaignID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCACampaigns, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cacampaign.Campaign
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
