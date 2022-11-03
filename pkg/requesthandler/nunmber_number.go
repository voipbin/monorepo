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

// NumberV1NumberGet sends the /v1/numbers/<number> GET request to number-manager
func (r *requestHandler) NumberV1NumberGetByNumber(ctx context.Context, num string) (*nmnumber.Number, error) {

	uri := fmt.Sprintf("/v1/numbers/%s", url.QueryEscape(num))

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get number .status: %d", tmp.StatusCode)
	}

	res := nmnumber.Number{}
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// NumberV1NumberGet sends a request to number-manager
// to get a given id of number.
// Returns number
func (r *requestHandler) NumberV1NumberGet(ctx context.Context, numberID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", numberID)

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// NumberV1NumberGets sends a request to number-manager
// to get a list of numbers.
// Returns list of numbers
func (r *requestHandler) NumberV1NumberGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// NumberV1NumberFlowDelete sends a request to number-manager
// to delete a flow from the number.
func (r *requestHandler) NumberV1NumberFlowDelete(ctx context.Context, flowID uuid.UUID) error {
	uri := fmt.Sprintf("/v1/number_flows/%s", flowID)

	tmp, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return err
	case tmp == nil:
		// not found
		return fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	return nil
}

// NMNumberCreate sends a request to the number-manager
// to create an number.
// Returns created number
func (r *requestHandler) NumberV1NumberCreate(ctx context.Context, customerID uuid.UUID, num string, callFlowID, messageFlowID uuid.UUID, name, detail string) (*nmnumber.Number, error) {
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

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceNumberNumbers, 15000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// NumberV1NumberDelete sends a request to number-manager
// to delete a number.
// Returns deletedd number
func (r *requestHandler) NumberV1NumberDelete(ctx context.Context, id uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceNumberNumbers, 15000, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// NumberV1NumberUpdate sends a request to the number-manager
// to update a number.
// Returns updated number info
func (r *requestHandler) NumberV1NumberUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	data := &nmrequest.V1DataNumbersIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// NumberV1NumberUpdate sends a request to the number-manager
// to update a number.
// Returns updated number info
func (r *requestHandler) NumberV1NumberUpdateFlowID(ctx context.Context, id, callFlowID, messageFlowID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s/flow_ids", id)

	data := &nmrequest.V1DataNumbersIDFlowIDPut{
		CallFlowID:    callFlowID,
		MessageFlowID: messageFlowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestNumber(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceNumberNumbers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData nmnumber.Number
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}
