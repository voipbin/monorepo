package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	omrequest "gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// OutdialV1OutdialCreate creates a new outdial.
// ctx: context
// customerID: customer id
// campaignID: campaign id
// name: name
// detail: detail
// data: data
func (r *requestHandler) OutdialV1OutdialCreate(ctx context.Context, customerID, campaignID uuid.UUID, name, detail, data string) (*omoutdial.Outdial, error) {

	uri := "/v1/outdials"

	reqData := &omrequest.V1DataOutdialsPost{
		CustomerID: customerID,
		CampaignID: campaignID,
		Name:       name,
		Detail:     detail,
		Data:       data,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialGetsByCustomerID sends a request to outdial-manager
// to get a list of outdials.
// Returns list of outdials
func (r *requestHandler) OutdialV1OutdialGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []omoutdial.Outdial
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// OutdialV1OutdialGet returns an outdial
// ctx: context
func (r *requestHandler) OutdialV1OutdialGet(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error) {

	uri := fmt.Sprintf("/v1/outdials/%s", outdialID)

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialDelete sends a request to outdial-manager
// to deleting the outdial.
func (r *requestHandler) OutdialV1OutdialDelete(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials/%s", outdialID)

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialUpdateBasicInfo sends a request to outdial-manager
// to update the outdial's basic info.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OutdialV1OutdialUpdateBasicInfo(ctx context.Context, outdialID uuid.UUID, name, detail string) (*omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials/%s", outdialID)

	data := &omrequest.V1DataOutdialsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialUpdateCampaignID sends a request to outdial-manager
// to update the outdial's campaign_id.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OutdialV1OutdialUpdateCampaignID(ctx context.Context, outdialID, campaignID uuid.UUID) (*omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials/%s/campaign_id", outdialID)

	data := &omrequest.V1DataOutdialsIDCampaignIDPut{
		CampaignID: campaignID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// OutdialV1OutdialUpdateCampaignID sends a request to outdial-manager
// to update the outdial's campaign_id.
// it returns updated outdial info if it succeed.
func (r *requestHandler) OutdialV1OutdialUpdateData(ctx context.Context, outdialID uuid.UUID, data string) (*omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials/%s/data", outdialID)

	tmpData := &omrequest.V1DataOutdialsIDDataPut{
		Data: data,
	}

	m, err := json.Marshal(tmpData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceOutdialOutdials, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res omoutdial.Outdial
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
