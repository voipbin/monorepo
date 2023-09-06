package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
	bmbilling "gitlab.com/voipbin/bin-manager/billing-manager.git/models/billing"
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
func (r *requestHandler) BillingV1AccountCreate(ctx context.Context, custoerID uuid.UUID, name string, detail string, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error) {
	uri := "/v1/accounts"

	m, err := json.Marshal(bmrequest.V1DataAccountsPOST{
		CustomerID:    custoerID,
		Name:          name,
		Detail:        detail,
		PaymentType:   paymentType,
		PaymentMethod: paymentMethod,
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

// BillingV1AccountUpdateBasicInfo updates a billing account's basic info.
func (r *requestHandler) BillingV1AccountUpdateBasicInfo(ctx context.Context, accountID uuid.UUID, name string, detail string) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	m, err := json.Marshal(bmrequest.V1DataAccountsIDPUT{
		Name:   name,
		Detail: detail,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPut, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// BillingV1AccountUpdatePaymentInfo updates a billing account's payment info.
func (r *requestHandler) BillingV1AccountUpdatePaymentInfo(ctx context.Context, accountID uuid.UUID, paymentType bmaccount.PaymentType, paymentMethod bmaccount.PaymentMethod) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s/payment_info", accountID)

	m, err := json.Marshal(bmrequest.V1DataAccountsIDPaymentInfoPUT{
		PaymentType:   paymentType,
		PaymentMethod: paymentMethod,
	})
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPut, "billing/accounts/<account-id>/payment_info", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// BillingV1AccountDelete deletes a billing account.
func (r *requestHandler) BillingV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodDelete, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// BillingV1AccountIsValidBalance returns true if the given account has valid balance for the given billing type
func (r *requestHandler) BillingV1AccountIsValidBalance(ctx context.Context, accountID uuid.UUID, billingType bmbilling.ReferenceType, country string, count int) (bool, error) {
	uri := fmt.Sprintf("/v1/accounts/%s/is_valid_balance", accountID)

	m, err := json.Marshal(bmrequest.V1DataAccountsIDIsValidBalancePOST{
		BillingType: string(billingType),
		Country:     country,
		Count:       count,
	})
	if err != nil {
		return false, err
	}

	tmp, err := r.sendRequestBilling(ctx, uri, rabbitmqhandler.RequestMethodPost, "billing/accounts/<account-id>/is_valid_balance", requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return false, err
	case tmp == nil:
		return false, fmt.Errorf("could not get response")
	case tmp.StatusCode > 299:
		return false, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var resData bmresponse.V1ResponseAccountsIDIsValidBalance
	if err := json.Unmarshal([]byte(tmp.Data), &resData); err != nil {
		return false, err
	}

	return resData.Valid, nil
}
