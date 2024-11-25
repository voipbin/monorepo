package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"monorepo/bin-common-handler/models/sock"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
	csrequest "monorepo/bin-customer-manager/pkg/listenhandler/models/request"
	"net/url"

	"github.com/gofrs/uuid"
)

// CustomerV1AccesskeyGet sends a request to customer-manager
// to getting a accesskey info.
// it returns accesskey info if it succeed.
func (r *requestHandler) CustomerV1AccesskeyGet(ctx context.Context, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error) {
	uri := fmt.Sprintf("/v1/accesskeys/%s", accesskeyID)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodGet, "customer/accesskeys", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res csaccesskey.Accesskey
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CustomerV1AccesskeyGets sends a request to customer-manager
// to getting a list of accesskeys info.
// it returns list of accesskeys info if it succeed.
func (r *requestHandler) CustomerV1AccesskeyGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]csaccesskey.Accesskey, error) {
	uri := fmt.Sprintf("/v1/accesskeys?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodGet, "customer/accesskeys", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []csaccesskey.Accesskey
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// // CustomerV1AccesskeyGetsByCustomerID sends a request to customer-manager
// // to getting a list of accesskeys info.
// // it returns list of accesskeys info if it succeed.
// func (r *requestHandler) CustomerV1AccesskeyGetsByCustomerID(ctx context.Context, pageToken string, pageSize uint64, customerID uuid.UUID) ([]csaccesskey.Accesskey, error) {
// 	uri := fmt.Sprintf("/v1/accesskeys?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

// 	// parse filters
// 	filters := map[string]string{
// 		"deleted":     "false",
// 		"customer_id": customerID.String(),
// 	}
// 	uri = r.utilHandler.URLMergeFilters(uri, filters)

// 	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodGet, "customer/accesskeys", requestTimeoutDefault, 0, ContentTypeJSON, nil)
// 	switch {
// 	case err != nil:
// 		return nil, err
// 	case tmp == nil:
// 		// not found
// 		return nil, fmt.Errorf("response code: %d", 404)
// 	case tmp.StatusCode > 299:
// 		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
// 	}

// 	var res []csaccesskey.Accesskey
// 	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
// 		return nil, err
// 	}

// 	return res, nil
// }

// CustomerV1AccesskeyCreate sends a request to customer-manager
// to creating a new accesskey info.
// it returns created accesskey info if it succeed.
// expire: seconds
func (r *requestHandler) CustomerV1AccesskeyCreate(ctx context.Context, customerID uuid.UUID, name string, detail string, expire int32) (*csaccesskey.Accesskey, error) {
	uri := "/v1/accesskeys"

	reqData := csrequest.V1DataAccesskeysPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		Expire:     expire,
	}

	m, err := json.Marshal(reqData)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPost, "customer/acceskeys", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res csaccesskey.Accesskey
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CustomerV1AccesskeyDelete sends a request to customer-manager
// to deleting the accesskey.
// it returns deleted accesskey if it succeed.
func (r *requestHandler) CustomerV1AccesskeyDelete(ctx context.Context, accesskeyID uuid.UUID) (*csaccesskey.Accesskey, error) {
	uri := fmt.Sprintf("/v1/accesskeys/%s", accesskeyID)

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodDelete, "customer/accesskeys", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res csaccesskey.Accesskey
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// CustomerV1AccesskeyUpdate sends a request to customer-manager
// to update the detail accesskey info.
func (r *requestHandler) CustomerV1AccesskeyUpdate(ctx context.Context, accesskeyID uuid.UUID, name string, detail string) (*csaccesskey.Accesskey, error) {
	uri := fmt.Sprintf("/v1/accesskeys/%s", accesskeyID)

	data := &csrequest.V1DataAccesskeysIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(ctx, uri, sock.RequestMethodPut, "customer/accesskeys", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res csaccesskey.Accesskey
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
