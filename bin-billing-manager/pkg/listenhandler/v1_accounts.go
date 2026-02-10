package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/account"
	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/listenhandler/models/request"
	"monorepo/bin-billing-manager/pkg/listenhandler/models/response"
)

// processV1AccountsIDGet handles GET /v1/accounts/<account-id> request
func (h *listenHandler) processV1AccountsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.accountHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDPut handles PUT /v1/accounts/<account-id> request
func (h *listenHandler) processV1AccountsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDPUT
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	c, err := h.accountHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not get account info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDBalanceAddForcePost handles POST /v1/accounts/<account-id>/balance_add_force request
func (h *listenHandler) processV1AccountsIDBalanceAddForcePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDBalanceAddForcePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDBalanceAddForcePOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	c, err := h.accountHandler.AddBalance(ctx, id, req.Balance)
	if err != nil {
		log.Errorf("Could not add the balance. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDBalanceSubtractForcePost handles POST /v1/accounts/<account-id>/balance_subtract_force request
func (h *listenHandler) processV1AccountsIDBalanceSubtractForcePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDBalanceSubtractForcePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDBalanceSubtractForcePOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	c, err := h.accountHandler.SubtractBalance(ctx, id, req.Balance)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDIsValidBalancePost handles POST /v1/accounts/<account-id>/is_valid_balance request
func (h *listenHandler) processV1AccountsIDIsValidBalancePost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDIsValidBalancePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDIsValidBalancePOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	valid, err := h.accountHandler.IsValidBalance(ctx, accountID, billing.ReferenceType(req.BillingType), req.Country, req.Count)
	if err != nil {
		log.Errorf("Could not validate the account's balance info. err: %v", err)
		return simpleResponse(404), nil
	}

	tmp := &response.V1ResponseAccountsIDIsValidBalance{
		Valid: valid,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDIsValidResourceLimitPost handles POST /v1/accounts/<account-id>/is_valid_resource_limit request
func (h *listenHandler) processV1AccountsIDIsValidResourceLimitPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDIsValidResourceLimitPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDIsValidResourceLimitPOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	valid, err := h.accountHandler.IsValidResourceLimit(ctx, accountID, account.ResourceType(req.ResourceType))
	if err != nil {
		log.Errorf("Could not validate the account's resource limit. err: %v", err)
		return simpleResponse(404), nil
	}

	tmp := &response.V1ResponseAccountsIDIsValidResourceLimit{
		Valid: valid,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIsValidBalanceByCustomerIDPost handles POST /v1/accounts/is_valid_balance_by_customer_id request
func (h *listenHandler) processV1AccountsIsValidBalanceByCustomerIDPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIsValidBalanceByCustomerIDPost",
		"request": m,
	})

	var req request.V1DataAccountsIsValidBalanceByCustomerIDPOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)

	valid, err := h.accountHandler.IsValidBalanceByCustomerID(ctx, customerID, billing.ReferenceType(req.BillingType), req.Country, req.Count)
	if err != nil {
		log.Errorf("Could not validate the account's balance info. err: %v", err)
		return simpleResponse(404), nil
	}

	tmp := &response.V1ResponseAccountsIDIsValidBalance{
		Valid: valid,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIsValidResourceLimitByCustomerIDPost handles POST /v1/accounts/is_valid_resource_limit_by_customer_id request
func (h *listenHandler) processV1AccountsIsValidResourceLimitByCustomerIDPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIsValidResourceLimitByCustomerIDPost",
		"request": m,
	})

	var req request.V1DataAccountsIsValidResourceLimitByCustomerIDPOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)

	valid, err := h.accountHandler.IsValidResourceLimitByCustomerID(ctx, customerID, account.ResourceType(req.ResourceType))
	if err != nil {
		log.Errorf("Could not validate the account's resource limit. err: %v", err)
		return simpleResponse(404), nil
	}

	tmp := &response.V1ResponseAccountsIDIsValidResourceLimit{
		Valid: valid,
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDPaymentInfoPut handles PUT /v1/accounts/<account-id>/payment_info request
func (h *listenHandler) processV1AccountsIDPaymentInfoPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDPaymentInfoPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	accountID := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDPaymentInfoPUT
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.accountHandler.UpdatePaymentInfo(ctx, accountID, req.PaymentType, req.PaymentMethod)
	if err != nil {
		log.Errorf("Could not update the account's payment info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
