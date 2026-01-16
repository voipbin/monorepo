package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	omoutdial "monorepo/bin-outdial-manager/models/outdial"
	omrequest "monorepo/bin-outdial-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
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

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodPost, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// OutdialV1OutdialList sends a request to outdial-manager
// to get a list of outdials.
// Returns list of outdials
func (r *requestHandler) OutdialV1OutdialList(ctx context.Context, pageToken string, pageSize uint64, filters map[omoutdial.Field]any) ([]omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodGet, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// OutdialV1OutdialGet returns an outdial
// ctx: context
func (r *requestHandler) OutdialV1OutdialGet(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error) {

	uri := fmt.Sprintf("/v1/outdials/%s", outdialID)

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodGet, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// OutdialV1OutdialDelete sends a request to outdial-manager
// to deleting the outdial.
func (r *requestHandler) OutdialV1OutdialDelete(ctx context.Context, outdialID uuid.UUID) (*omoutdial.Outdial, error) {
	uri := fmt.Sprintf("/v1/outdials/%s", outdialID)

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodDelete, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodPut, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodPut, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestOutdial(ctx, uri, sock.RequestMethodPut, "outdial/outdials", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res omoutdial.Outdial
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
