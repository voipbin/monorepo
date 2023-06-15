package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	bmresponse "gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/response"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// BillingV1AccountGets returns list of billing accounts.
func (r *requestHandler) BillingV1AccountGets(ctx context.Context, pageToken string, pageSize uint64) ([]bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodGet, "billing/accounts", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []bmaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the response data")
	}

	return res, nil
}

// BillingV1AccountGet returns a billing account.
func (r *requestHandler) BillingV1AccountGet(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodGet, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res bmaccount.Account
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the response data")
	}

	return &res, nil
}

// BillingV1AccountIsValidBalanceByCustomerID returns true if the given customer's billing account has enough balance.
func (r *requestHandler) BillingV1AccountIsValidBalanceByCustomerID(ctx context.Context, customerID uuid.UUID) (bool, error) {
	uri := fmt.Sprintf("/v1/accounts/customer_id/%s/is_valid_balance", customerID)

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPost, "billing/accounts/customer_id/<customer-id>/is_valid_balance", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return false, err
	case tmp == nil:
		return false, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return false, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	res := bmresponse.V1ResponseAccountsIDIsValidBalance{}
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return false, errors.Wrap(err, "could not unmarshal the response data")
	}

	return res.Valid, nil
}

// BillingV1AccountGetByCustomerID returns given customer id's billing account
func (r *requestHandler) BillingV1AccountGetByCustomerID(ctx context.Context, customerID uuid.UUID) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/customer_id/%s", customerID)

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodGet, "billing/accounts/customer_id/<customer-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		return nil, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	res := bmaccount.Account{}
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal the response data")
	}

	return &res, nil
}
