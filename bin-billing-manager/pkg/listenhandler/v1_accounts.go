package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-billing-manager/pkg/listenhandler/models/request"
	"monorepo/bin-billing-manager/pkg/listenhandler/models/response"
)

// processV1AccountsGet handles GET /v1/accounts request
func (h *listenHandler) processV1AccountsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters
	filters := h.utilHandler.URLParseFilters(u)

	as, err := h.accountHandler.Gets(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get accounts info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(as)
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

// processV1AccountsPost handles POST /v1/accounts request
func (h *listenHandler) processV1AccountsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsPost",
		"request": m,
	})

	var req request.V1DataAccountsPOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	tmp, err := h.accountHandler.Create(ctx, req.CustomerID, req.Name, req.Detail, req.PaymentType, req.PaymentMethod)
	if err != nil {
		log.Errorf("Could not create account info. err: %v", err)
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

// processV1AccountsIDDelete handles DELETE /v1/accounts/<account-id> request
func (h *listenHandler) processV1AccountsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	c, err := h.accountHandler.Delete(ctx, id)
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

// // processV1AccountsCustomerIDIDGet handles GET /v1/accounts/customer_id/<cusotmer-id> request
// func (h *listenHandler) processV1AccountsCustomerIDIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":    "processV1AccountsCustomerIDIDGet",
// 		"request": m,
// 	})

// 	uriItems := strings.Split(m.URI, "/")
// 	if len(uriItems) < 4 {
// 		return simpleResponse(400), nil
// 	}

// 	customerID := uuid.FromStringOrNil(uriItems[4])

// 	c, err := h.accountHandler.GetByCustomerID(ctx, customerID)
// 	if err != nil {
// 		log.Errorf("Could not get account info. err: %v", err)
// 		return simpleResponse(404), nil
// 	}

// 	data, err := json.Marshal(c)
// 	if err != nil {
// 		return simpleResponse(404), nil
// 	}

// 	res := &sock.Response{
// 		StatusCode: 200,
// 		DataType:   "application/json",
// 		Data:       data,
// 	}

// 	return res, nil
// }

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
