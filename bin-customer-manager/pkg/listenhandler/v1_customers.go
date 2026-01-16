package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-customer-manager/pkg/listenhandler/models/request"
	"monorepo/bin-customer-manager/pkg/listenhandler/models/response"
)

// processV1CustomersGet handles GET /v1/customers request
func (h *listenHandler) processV1CustomersGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersGet",
		"request": m,
	})
	log.Debug("Executing processV1CustomersGet.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	filters, err := utilhandler.ConvertFilters[customer.FieldStruct, customer.Field](customer.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get customers info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersPost handles Post /v1/customers request
func (h *listenHandler) processV1CustomersPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersPost",
		"request": m,
	})
	log.Debug("Executing processV1CustomersPost.")

	var reqData request.V1DataCustomersPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log.Debug("Creating a customer.")

	tmp, err := h.customerHandler.Create(
		ctx,
		reqData.Name,
		reqData.Detail,
		reqData.Email,
		reqData.PhoneNumber,
		reqData.Address,
		reqData.WebhookMethod,
		reqData.WebhookURI,
	)
	if err != nil {
		log.Errorf("Could not create the customer info. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDGet handles Get /v1/customers/<customer-id> request
func (h *listenHandler) processV1CustomersIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDGet",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDGet.")

	tmp, err := h.customerHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not update the customer info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDDelete handles Delete /v1/customers/<user-id> request
func (h *listenHandler) processV1CustomersIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDDelete",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDDelete.")

	tmp, err := h.customerHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the customer info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDPut handles Put /v1/customers/<customer-id> request
func (h *listenHandler) processV1CustomersIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDPut",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDPut.")

	var reqData request.V1DataCustomersIDPut
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		// same call-id is already exsit
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdateBasicInfo(
		ctx,
		id,
		reqData.Name,
		reqData.Detail,
		reqData.Email,
		reqData.PhoneNumber,
		reqData.Address,
		reqData.WebhookMethod,
		reqData.WebhookURI,
	)
	if err != nil {
		log.Errorf("Could not update the user info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDIsValidBalance handles Put /v1/customers/<customer-id>/is_valid_balance request
func (h *listenHandler) processV1CustomersIDIsValidBalance(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1CustomersIDIsValidBalance",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debug("Executing processV1CustomersIDIsValidBalance.")

	var req request.V1DataCustomersIDIsValidBalancePost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	valid, err := h.customerHandler.IsValidBalance(ctx, id, bmbilling.ReferenceType(req.ReferenceType), req.Country, req.Count)
	if err != nil {
		log.Errorf("Could not update the customer's permission ids. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp := &response.V1ResponseCustomersIDIsValidBalancePost{
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

// processV1CustomersIDBillingAccountIDPut handles Put /v1/customers/<customer-id>/billing_account_id request
func (h *listenHandler) processV1CustomersIDBillingAccountIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":        "processV1CustomersIDBillingAccountIDPut",
		"customer_id": id,
	})
	log.Debug("Executing processV1CustomersIDBillingAccountIDPut.")

	var req request.V1DataCustomersIDBillingAccountIDPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdateBillingAccountID(ctx, id, req.BillingAccountID)
	if err != nil {
		log.Errorf("Could not update the customer's permission ids. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
