package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	nmrequest "gitlab.com/voipbin/bin-manager/number-manager.git/pkg/listenhandler/models/request"
)

// NMNumberCreate sends a request to the number-manager
// to create an number.
// Returns created number
func (r *requestHandler) NMNumberCreate(userID uint64, numb string) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers")

	data := &nmrequest.V1DataNumbersPost{
		UserID: userID,
		Number: numb,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodPost, resourceNumberNumbers, 15, 0, ContentTypeJSON, m)
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

// NMNumberGets sends a request to number-manager
// to get a list of numbers.
// Returns list of numbers
func (r *requestHandler) NMNumberGets(userID uint64, pageToken string, pageSize uint64) ([]nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers?page_token=%s&page_size=%d&user_id=%d", url.QueryEscape(pageToken), pageSize, userID)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15, 0, ContentTypeJSON, nil)
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

// NMNumberGet sends a request to number-manager
// to get a given id of number.
// Returns number
func (r *requestHandler) NMNumberGet(numberID uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", numberID)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodGet, resourceNumberNumbers, 15, 0, ContentTypeJSON, nil)
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

// NMNumberDelete sends a request to number-manager
// to delete a number.
// Returns deletedd number
func (r *requestHandler) NMNumberDelete(id uuid.UUID) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", id)

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodDelete, resourceNumberNumbers, 15, 0, ContentTypeJSON, nil)
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

// NMNumberUpdate sends a request to the number-manager
// to update a number.
// Returns updated number info
func (r *requestHandler) NMNumberUpdate(num *nmnumber.Number) (*nmnumber.Number, error) {
	uri := fmt.Sprintf("/v1/numbers/%s", num.ID)

	data := &nmrequest.V1DataNumbersIDPut{
		FlowID: num.FlowID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestNumber(uri, rabbitmqhandler.RequestMethodPut, resourceNumberNumbers, 15, 0, ContentTypeJSON, m)
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
