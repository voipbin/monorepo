package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"
	cscustomer "monorepo/bin-customer-manager/models/customer"
	csrequest "monorepo/bin-customer-manager/pkg/listenhandler/models/request"
	csresponse "monorepo/bin-customer-manager/pkg/listenhandler/models/response"

	"github.com/gofrs/uuid"
)

// CustomerV1CustomerGet sends a request to customer-manager
// to getting a detail customer info.
// it returns detail customer info if it succeed.
func (r *requestHandler) CustomerV1CustomerGet(ctx context.Context, customerID uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", customerID)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodGet, "customer/customers", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CustomerV1CustomerGets sends a request to customer-manager
// to getting a list of customers info.
// it returns detail customer info if it succeed.
func (r *requestHandler) CustomerV1CustomerGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodGet, "customer/customers", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// CustomerV1CustomerCreate sends the request to create the customer
// requestTimeout: milliseconds
func (r *requestHandler) CustomerV1CustomerCreate(
	ctx context.Context,
	requestTimeout int,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.Customer, error) {
	uri := "/v1/customers"

	reqData := csrequest.V1DataCustomersPost{
		Name:          name,
		Detail:        detail,
		Email:         email,
		PhoneNumber:   phoneNumber,
		Address:       address,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil

}

// CustomerV1CustomerDelete sends the request to delete the customer
func (r *requestHandler) CustomerV1CustomerDelete(ctx context.Context, id uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", id)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodDelete, "customer/customers", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil

}

// CustomerV1CustomerUpdate sends a request to customer-manager
// to update the detail customer info.
func (r *requestHandler) CustomerV1CustomerUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	email string,
	phoneNumber string,
	address string,
	webhookMethod cscustomer.WebhookMethod,
	webhookURI string,
) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s", id)

	data := &csrequest.V1DataCustomersIDPut{
		Name:          name,
		Detail:        detail,
		Email:         email,
		PhoneNumber:   phoneNumber,
		Address:       address,
		WebhookMethod: webhookMethod,
		WebhookURI:    webhookURI,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPut, "customer/customers", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// CustomerV1CustomerIsValidBalance sends a request to customer-manager
// returns true if the customer has valid balance.
func (r *requestHandler) CustomerV1CustomerIsValidBalance(ctx context.Context, customerID uuid.UUID, referenceType bmbilling.ReferenceType, country string, count int) (bool, error) {
	uri := fmt.Sprintf("/v1/customers/%s/is_valid_balance", customerID)

	data := &csrequest.V1DataCustomersIDIsValidBalancePost{
		ReferenceType: referenceType,
		Country:       country,
		Count:         count,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/customers/<customer-id>/is_valid_balance", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return false, err
	}

	var res csresponse.V1ResponseCustomersIDIsValidBalancePost
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return false, errParse
	}

	return res.Valid, nil
}

// CustomerV1CustomerUpdateBillingAccountID sends a request to customer-manager
// to update the customer's billing account id.
func (r *requestHandler) CustomerV1CustomerUpdateBillingAccountID(ctx context.Context, customerID uuid.UUID, biillingAccountID uuid.UUID) (*cscustomer.Customer, error) {
	uri := fmt.Sprintf("/v1/customers/%s/billing_account_id", customerID)

	data := &csrequest.V1DataCustomersIDBillingAccountIDPut{
		BillingAccountID: biillingAccountID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPut, "customer/customers/<customer-id>/billing_account_id", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cscustomer.Customer
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
