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

// processV1OrderNumbersPost handles POST /v1/order_numbers request
func (h *listenHandler) processV1OrderNumbersPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var reqData request.V1DataOrderNumbersPost
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		// same call-id is already exsit
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}
	log := logrus.WithFields(
		logrus.Fields{
			"user":    reqData.UserID,
			"numbers": reqData.Numbers,
		},
	)

	numbers, err := h.numberHandler.CreateOrderNumbers(reqData.UserID, reqData.Numbers)
	if err != nil {
		log.Errorf("Could not handle the order number. err: %v", err)
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

// processV1OrderNumbersIDDelete handles DELETE /v1/order_numbers/<id> request
func (h *listenHandler) processV1OrderNumbersIDDelete(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
	number, err := h.numberHandler.ReleaseOrderNumbers(ctx, id)
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

// processV1OrderNumbersIDGet handles GET /v1/order_numbers/<id> request
func (h *listenHandler) processV1OrderNumbersIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
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
	number, err := h.numberHandler.GetOrderNumber(ctx, id)
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

// processV1OrderNumbersGet handles GET /v1/order_numbers request
func (h *listenHandler) processV1OrderNumbersGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	u, err := url.Parse(req.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get user_id
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	log := logrus.WithFields(logrus.Fields{
		"user":  userID,
		"size":  pageSize,
		"token": pageToken,
	})

	ctx := context.Background()
	numbers, err := h.numberHandler.GetOrderNumbers(ctx, userID, pageSize, pageToken)
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
