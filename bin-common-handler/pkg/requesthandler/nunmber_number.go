package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	nmnumber "monorepo/bin-number-manager/models/number"
	nmrequest "monorepo/bin-number-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// NumberV1NumberGet sends a request to number-manager
// to get a given id of number.
// Returns number
func (r *requestHandler) NumberV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", numberID)

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodGet, "number/numbers", 15000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// NumberV1NumberList sends a request to number-manager
// to get a list of numbers.
// Returns list of numbers
func (r *requestHandler) NumberV1NumberList(ctx context.Context, pageToken string, pageSize uint64, filters map[nmnumber.Field]any) ([]nmnumber.Number, error) {

	uri := fmt.Sprintf("/v1/numbers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodGet, "number/numbers", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPost, "number/numbers", 15000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// NumberV1NumberDelete sends a request to number-manager
// to delete a number.
// Returns deletedd number
func (r *requestHandler) NumberV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodDelete, "number/numbers", 15000, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPut, "number/numbers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPut, "number/numbers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

type virtualNumberCountByCustomerRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

type virtualNumberCountByCustomerResponse struct {
	Count int `json:"count"`
}

// NumberV1VirtualNumberCountByCustomerID sends a request to number-manager
// to get the count of virtual numbers for the given customer.
func (r *requestHandler) NumberV1VirtualNumberCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error) {
	uri := "/v1/numbers/count_virtual_by_customer"

	m, err := json.Marshal(virtualNumberCountByCustomerRequest{
		CustomerID: customerID,
	})
	if err != nil {
		return 0, err
	}

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodGet, "number/numbers/count_virtual_by_customer", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return 0, err
	}

	var res virtualNumberCountByCustomerResponse
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return 0, errParse
	}

	return res.Count, nil
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestNumber(ctx, uri, sock.RequestMethodPost, "number/numbers/renew", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []nmnumber.Number
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}
