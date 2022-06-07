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

	"gitlab.com/voipbin/bin-manager/customer-manager.git/pkg/listenhandler/models/request"
)

// processV1CustomersGet handles GET /v1/customers request
func (h *listenHandler) processV1CustomersGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	log := logrus.WithFields(logrus.Fields{
		"size":  pageSize,
		"token": pageToken,
	})

	tmp, err := h.customerHandler.Gets(ctx, pageSize, pageToken)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersPost handles Post /v1/customers request
func (h *listenHandler) processV1CustomersPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1CustomersPost",
		})
	log.Debug("Executing processV1CustomersPost.")

	var reqData request.V1DataCustomersPost
	if err := json.Unmarshal([]byte(m.Data), &reqData); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log = log.WithFields(logrus.Fields{
		"username":       reqData.Username,
		"permission_ids": reqData.PermissionIDs,
	})
	log.Debug("Creating a customer.")

	tmp, err := h.customerHandler.Create(ctx, reqData.Username, reqData.Password, reqData.Name, reqData.Detail, reqData.WebhookMethod, reqData.WebhookURI, reqData.PermissionIDs)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDGet handles Get /v1/customers/<customer-id> request
func (h *listenHandler) processV1CustomersIDGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDDelete handles Delete /v1/customers/<user-id> request
func (h *listenHandler) processV1CustomersIDDelete(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDPut handles Put /v1/customers/<customer-id> request
func (h *listenHandler) processV1CustomersIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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

	tmp, err := h.customerHandler.UpdateBasicInfo(ctx, id, reqData.Name, reqData.Detail, reqData.WebhookMethod, reqData.WebhookURI)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDPasswordPut handles Put /v1/customers/<customer-id>/password request
func (h *listenHandler) processV1CustomersIDPasswordPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDPasswordPut",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDPasswordPut.")

	var req request.V1DataCustomersIDPasswordPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdatePassword(ctx, id, req.Password)
	if err != nil {
		log.Errorf("Could not update the customer's password. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDPermissionIDsPut handles Put /v1/customers/<customer-id>/permission request
func (h *listenHandler) processV1CustomersIDPermissionIDsPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDPermissionPut",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDPermissionPut.")

	var req request.V1DataCustomersIDPermissionIDsPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdatePermissionIDs(ctx, id, req.PermissionIDs)
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

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1CustomersIDLineInfoPut handles Put /v1/customers/<customer-id>/line_info request
func (h *listenHandler) processV1CustomersIDLineInfoPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"func":        "processV1CustomersIDLineInfoPut",
			"customer_id": id,
		})
	log.Debug("Executing processV1CustomersIDLineInfoPut.")

	var req request.V1DataCustomersIDLineInfoPut
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.customerHandler.UpdateLineInfo(ctx, id, req.LineSecret, req.LineToken)
	if err != nil {
		log.Errorf("Could not update the customer's line info. err: %v", err)
		return simpleResponse(400), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the result data. data: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}
	log.Debugf("Sending result: %v", data)

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
