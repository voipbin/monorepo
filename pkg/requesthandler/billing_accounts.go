package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	bmrequest "gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/request"
	bmresponse "gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/response"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// BillingV1AccountGets returns list of billing accounts.
func (r *requestHandler) BillingV1AccountGets(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?customer_id=%s&page_token=%s&page_size=%d", customerID, url.QueryEscape(pageToken), pageSize)

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

// BillingV1AccountCreate creates a new billing account.
func (r *requestHandler) BillingV1AccountCreate(ctx context.Context, custoerID uuid.UUID, name string, detail string) (*bmaccount.Account, error) {
	uri := "/v1/accounts"

	m, err := json.Marshal(bmrequest.V1DataAccountsPOST{
		CustomerID: custoerID,
		Name:       name,
		Detail:     detail,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPost, "billing/accounts", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// BillingV1AccountAddBalanceForce adds the balance to the account in forcedly
func (r *requestHandler) BillingV1AccountAddBalanceForce(ctx context.Context, accountID uuid.UUID, balance float32) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s/balance_add_force", accountID)

	m, err := json.Marshal(bmrequest.V1DataAccountsIDBalanceAddForcePOST{
		Balance: balance,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPost, "billing/accounts/<account-id>/balance_add", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// BillingV1AccountSubtractBalanceForce subtracts the balance from the account in forcedly
func (r *requestHandler) BillingV1AccountSubtractBalanceForce(ctx context.Context, accountID uuid.UUID, balance float32) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s/balance_subtract_force", accountID)

	m, err := json.Marshal(bmrequest.V1DataAccountsIDBalanceSubtractForcePOST{
		Balance: balance,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPost, "billing/accounts/<account-id>/balance_subtract", requestTimeoutDefault, 0, ContentTypeJSON, m)
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
