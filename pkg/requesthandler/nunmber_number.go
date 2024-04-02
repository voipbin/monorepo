package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	nmrequest "gitlab.com/voipbin/bin-manager/number-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// NumberV1NumberGet sends a request to number-manager
// to get a given id of number.
// Returns number
func (r *requestHandler) NumberV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", numberID)

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberGets sends a request to number-manager
// to get a list of numbers.
// Returns list of numbers
func (r *requestHandler) NumberV1NumberGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = parseFilters(uri, filters)

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// NMNumberCreate sends a request to the number-manager
// to create an number.
// Returns created number
func (r *requestHandler) NumberV1NumberCreate(ctx context.Context, customerID uuid.UUID, num string, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.Number, error) {
	uri := "/v1/numbers"

	data := &nmrequest.V1DataNumbersPost{
		CustomerID:    customerID,
		Number:        num,
		CallFlowID:    callFlowID,
		MessageFlowID: messageFlowID,
		Name:          name,
		Detail:        detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceNumberNumbers, 15000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberDelete sends a request to number-manager
// to delete a number.
// Returns deletedd number
func (r *requestHandler) NumberV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberUpdate sends a request to the number-manager
// to update a number.
// Returns updated number info
func (r *requestHandler) NumberV1NumberUpdate(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID, name string, detail string) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	data := &nmrequest.V1DataNumbersIDPut{
		CallFlowID:    callFlowID,
		MessageFlowID: messageFlowID,
		Name:          name,
		Detail:        detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberUpdate sends a request to the number-manager
// to update a number.
// Returns updated number info
func (r *requestHandler) NumberV1NumberUpdateFlowID(ctx context.Context, id uuid.UUID, callFlowID uuid.UUID, messageFlowID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s/flow_ids", id)

	data := &nmrequest.V1DataNumbersIDFlowIDPut{
		CallFlowID:    callFlowID,
		MessageFlowID: messageFlowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberRenewByTmRenew sends a request to the number-manager
// to renew the numbers by tm_renew.
// Returns renewed number info
func (r *requestHandler) NumberV1NumberRenewByTmRenew(ctx context.Context, tmRenew string) ([]nmnumber.Number, error) {
	uri := "/v1/numbers/renew"

	data := &nmrequest.V1DataNumbersRenewPost{
		TMRenew: tmRenew,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// NumberV1NumberRenewByTmRenew sends a request to the number-manager
// to renew the numbers by days.
// Returns renewed number info
func (r *requestHandler) NumberV1NumberRenewByDays(ctx context.Context, days int) ([]nmnumber.Number, error) {
	uri := "/v1/numbers/renew"

	data := &nmrequest.V1DataNumbersRenewPost{
		Days: days,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// NumberV1NumberRenewByHours sends a request to the number-manager
// to renew the numbers by hours.
// Returns renewed number info
func (r *requestHandler) NumberV1NumberRenewByHours(ctx context.Context, hours int) ([]nmnumber.Number, error) {
	uri := "/v1/numbers/renew"

	data := &nmrequest.V1DataNumbersRenewPost{
		Hours: hours,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []nmnumber.Number
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}
