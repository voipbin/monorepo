package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	carequest "gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CAV1OutplanCreate creates a new outplan.
func (r *requestHandler) CAV1OutplanCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	source *cmaddress.Address,
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

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPost, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not find action")
	}

	var res caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1OutplanGets sends a request to campaign-manager
// to getting a list of outplans.
// it returns detail list of outplans if it succeed.
func (r *requestHandler) CAV1OutplanGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodGet, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// CAV1OutplanGet sends a request to campaign-manager
// to getting a detail outplan info.
// it returns detail outplan info if it succeed.
func (r *requestHandler) CAV1OutplanGet(ctx context.Context, id uuid.UUID) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", id)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodGet, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1OutplanDelete sends a request to campaign-manager
// to deleting the outplan.
func (r *requestHandler) CAV1OutplanDelete(ctx context.Context, outplanID uuid.UUID) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", outplanID)

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodDelete, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1OutplanUpdateBasicInfo sends a request to campaign-manager
// to update the outplan's basic info.
// delay millisecond
func (r *requestHandler) CAV1OutplanUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*caoutplan.Outplan, error) {
	uri := fmt.Sprintf("/v1/outplans/%s", id)

	data := &carequest.V1DataOutplansIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CAV1OutplanUpdateDialInfo sends a request to campaign-manager
// to update the outplan's dial info.
// delay millisecond
func (r *requestHandler) CAV1OutplanUpdateDialInfo(
	ctx context.Context,
	id uuid.UUID,
	source *cmaddress.Address,
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

	tmp, err := r.sendRequestCampaign(uri, rabbitmqhandler.RequestMethodPut, resourceCAOutplans, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res caoutplan.Outplan
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
