package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/nmnumber"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// NMOrderNumberCreate sends a request to the number-manager
// to create an order number.
// Returns created order number
func (r *requestHandler) NMOrderNumberCreate(userID uint64, numb string) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/order_numbers")

	data := &request.NMV1DataOrderNumbersPost{
		UserID: userID,
		Number: numb,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodPost, resourceNumberOrderNumbers, 15, 0, ContentTypeJSON, m)
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

// NMOrderNumberGets sends a request to number-manager
// to get a list of order numbers.
// Returns list of order numbers
func (r *requestHandler) NMOrderNumberGets(userID uint64, pageToken string, pageSize uint64) ([]nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/order_numbers?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberOrderNumbers, 15, 0, ContentTypeJSON, nil)
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

// NMOrderNumberGet sends a request to number-manager
// to get a given id of order number.
// Returns order number
func (r *requestHandler) NMOrderNumberGet(numberID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/order_numbers/%s", numberID)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberOrderNumbers, 15, 0, ContentTypeJSON, nil)
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

// NMOrderNumberDelete sends a request to number-manager
// to delete a ordered number.
// Returns deletedd order number
func (r *requestHandler) NMOrderNumberDelete(id uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/order_numbers/%s", id)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodDelete, resourceNumberOrderNumbers, 15, 0, ContentTypeJSON, nil)
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
