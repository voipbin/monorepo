package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	csrequest "gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CSV1CustomerGet sends a request to customer-manager
// to getting a detail customer info.
// it returns detail customer info if it succeed.
func (r *requestHandler) CSV1CustomerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", customerID)

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodGet, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CSV1CustomerGets sends a request to customer-manager
// to getting a list of customers info.
// it returns detail customer info if it succeed.
func (r *requestHandler) CSV1CustomerGets(ctx context.Context, pageToken string, pageSize uint64) ([]cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodGet, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData []cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return resData, nil
}

// CSV1CustomerCreate sends the request to create the customer
// requestTimeout: milliseconds
func (r *requestHandler) CSV1CustomerCreate(ctx context.Context, requestTimeout int, username, password, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string, permissionIDs []uuid.UUID) (*cscustomer.Customer, error) {
	uri := "/v1/customers"

	reqData := csrequest.V1DataCustomersPost{
		Username:      username,
		Password:      password,
		Name:          name,
		Detail:        detail,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,
		PermissionIDs: permissionIDs,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodPost, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CSV1CustomerDelete sends the request to delete the customer
func (r *requestHandler) CSV1CustomerDelete(ctx context.Context, id uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", id)

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodDelete, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CSV1CustomerUpdate sends a request to customer-manager
// to update the detail customer info.
func (r *requestHandler) CSV1CustomerUpdate(ctx context.Context, id uuid.UUID, name, detail string, webhookMethod cscustomer.WebhookMethod, webhookURI string) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", id)

	data := &csrequest.V1DataCustomersIDPut{
		Name:          name,
		Detail:        detail,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodPut, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CSV1CustomerUpdate sends a request to customer-manager
// to update the detail customer info.
// requestTimeout: milliseconds
func (r *requestHandler) CSV1CustomerUpdatePassword(ctx context.Context, requestTimeout int, id uuid.UUID, password string) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s/password", id)

	data := &csrequest.V1DataCustomersIDPasswordPut{
		Password: password,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodPut, resourceCSCustomers, requestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}

// CSV1CustomerUpdate sends a request to customer-manager
// to update the detail customer info.
func (r *requestHandler) CSV1CustomerUpdatePermissionIDs(ctx context.Context, id uuid.UUID, permissionIDs []uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s/permission_ids", id)

	data := &csrequest.V1DataCustomersIDPermissionIDsPut{
		PermissionIDs: permissionIDs,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	res, err := r.sendRequestCS(uri, rabbitmqhandler.RequestMethodPut, resourceCSCustomers, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case res == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case res.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", res.StatusCode)
	}

	var resData cscustomer.Customer
	if err := json.Unmarshal([]byte(res.Data), &resData); err != nil {
		return nil, err
	}

	return &resData, nil
}
