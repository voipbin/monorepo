package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	csrequest "gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CustomerV1Login sends a request to customer-manager
// to login.
// it returns customer if it succeed.
// timeout: milliseconds
func (r *requestHandler) CustomerV1Login(ctx context.Context, timeout int, username, password string) (*cscustomer.Customer, error) {
	uri := "/v1/login"

	req := &csrequest.V1DataLoginPost{
		Username: username,
		Password: password,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestCustomer(uri, rabbitmqhandler.RequestMethodPost, resourceCustomerLogin, timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cscustomer.Customer
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
