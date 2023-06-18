package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/response"
)

// processV1AccountsGet handles GET /v1/accounts request
func (h *listenHandler) processV1AccountsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	as, err := h.accountHandler.Gets(ctx, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(as)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDGet handles GET /v1/accounts/<account-id> request
func (h *listenHandler) processV1AccountsIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsCustomerIDIDGet handles GET /v1/accounts/customer_id/<cusotmer-id> request
func (h *listenHandler) processV1AccountsCustomerIDIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsCustomerIDIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(uriItems[4])

	c, err := h.accountHandler.GetByCustomerID(ctx, customerID)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsCustomerIDIDIsValidBalancePost handles GET /v1/accounts/customer_id/<customer-id>/is_valid_balance request
func (h *listenHandler) processV1AccountsCustomerIDIDIsValidBalancePost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsCustomerIDIDIsValidBalancePost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(uriItems[4])

	valid, err := h.accountHandler.IsValidBalanceByCustomerID(ctx, customerID)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDBalanceAddPost handles POST /v1/accounts/<account-id>/balance_add request
func (h *listenHandler) processV1AccountsIDBalanceAddPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDBalanceAddPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDBalanceAddPOST
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		return nil, err
	}

	c, err := h.accountHandler.AddBalance(ctx, id, req.Balance)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(c)
	if err != nil {
		return simpleResponse(404), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1AccountsIDBalanceSubtractPost handles POST /v1/accounts/<account-id>/balance_subtract request
func (h *listenHandler) processV1AccountsIDBalanceSubtractPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1AccountsIDBalanceSubtractPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	var req request.V1DataAccountsIDBalanceSubtractPOST
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
