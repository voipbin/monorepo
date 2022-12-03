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

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/listenhandler/models/request"
)

// processV1NumbersPost handles POST /v1/numbers request
func (h *listenHandler) processV1NumbersPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	var reqData request.V1DataNumbersPost
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}
	log := logrus.WithFields(
		logrus.Fields{
			"user":   reqData.CustomerID,
			"number": reqData.Number,
			"name":   reqData.Name,
			"detail": reqData.Detail,
		},
	)

	numb, err := h.numberHandler.Create(ctx, reqData.CustomerID, reqData.Number, reqData.CallFlowID, reqData.MessageFlowID, reqData.Name, reqData.Detail)
	if err != nil {
		log.Errorf("Could not handle the order number. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numb)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numb, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDDelete handles DELETE /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debugf("Executing processV1OrderNumbersIDDelete. number: %s", id)

	ctx := context.Background()
	number, err := h.numberHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not delete the number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDGet handles GET /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debugf("Executing processV1OrderNumbersIDGet. number: %s", id)

	ctx := context.Background()
	number, err := h.numberHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDPut handles PUT /v1/numbers/<id> request
func (h *listenHandler) processV1NumbersIDPut(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var reqData request.V1DataNumbersIDPut
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"func":      "processV1NumbersIDPut",
			"number_id": id,
		},
	)
	log.Debugf("Executing processV1NumbersIDPut. number: %s", id)

	number, err := h.numberHandler.UpdateBasicInfo(ctx, id, reqData.Name, reqData.Detail)
	if err != nil {
		log.Debugf("Could not update the number. number: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersNumberGet handles GET /v1/numbers/<number> request
func (h *listenHandler) processV1NumbersNumberGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	num := uriItems[3]
	log := logrus.WithFields(
		logrus.Fields{
			"num": num,
		})
	log.Debugf("Executing processV1OrderNumbersNumberGet. number: %s", num)

	ctx := context.Background()
	number, err := h.numberHandler.GetByNumber(ctx, num)
	if err != nil {
		log.Debugf("Could not get a number. number: %s, err: %v", num, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersGet handles GET /v1/numbers request
func (h *listenHandler) processV1NumbersGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	log := logrus.WithFields(logrus.Fields{
		"user":  customerID,
		"size":  pageSize,
		"token": pageToken,
	})

	ctx := context.Background()
	numbers, err := h.numberHandler.GetsByCustomerID(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get numbers. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(numbers)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", numbers, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1NumbersIDFlowIDsPut handles PUT /v1/numbers/<id>/flow_id request
func (h *listenHandler) processV1NumbersIDFlowIDsPut(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	var reqData request.V1DataNumbersIDFlowIDPut
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}

	log := logrus.WithFields(
		logrus.Fields{
			"func":            "processV1NumbersIDFlowIDPut",
			"number_id":       id,
			"call_flow_id":    reqData.CallFlowID,
			"message_flow_id": reqData.MessageFlowID,
		},
	)
	log.Debugf("Executing processV1NumbersIDPut. number: %s", id)

	number, err := h.numberHandler.UpdateFlowID(ctx, id, reqData.CallFlowID, reqData.MessageFlowID)
	if err != nil {
		log.Debugf("Could not update the number's flow_id. number_id: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(number)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", number, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
