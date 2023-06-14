package listenhandler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/billing-manager.git/pkg/listenhandler/models/response"
)

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
		log.Errorf("Could not get call info. err: %v", err)
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
