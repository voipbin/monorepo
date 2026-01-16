package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	bmrequest "monorepo/bin-billing-manager/pkg/listenhandler/models/request"
	bmresponse "monorepo/bin-billing-manager/pkg/listenhandler/models/response"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// BillingV1AccountGets returns list of billing accounts.
func (r *requestHandler) BillingV1AccountList(ctx context.Context, pageToken string, pageSize uint64, filters map[bmaccount.Field]any) ([]bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/accounts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/accounts", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// BillingV1AccountGet returns a billing account.
func (r *requestHandler) BillingV1AccountGet(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodGet, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPut, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPut, "billing/accounts/<account-id>/payment_info", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// BillingV1AccountDelete deletes a billing account.
func (r *requestHandler) BillingV1AccountDelete(ctx context.Context, accountID uuid.UUID) (*bmaccount.Account, error) {
	uri := fmt.Sprintf("/v1/accounts/%s", accountID)

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodDelete, "billing/accounts/<account-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/accounts/<account-id>/balance_add", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/accounts/<account-id>/balance_subtract", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res bmaccount.Account
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
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

	tmp, err := r.sendRequestBilling(ctx, uri, sock.RequestMethodPost, "billing/accounts/<account-id>/is_valid_balance", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return false, err
	}

	var res bmresponse.V1ResponseAccountsIDIsValidBalance
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return false, errParse
	}

	return res.Valid, nil
}
