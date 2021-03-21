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
	"gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/listenhandler/models/request"
)

// processV1NumbersPost handles POST /v1/numbers request
func (h *listenHandler) processV1NumbersPost(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var reqData request.V1DataNumbersPost
	if err := json.Unmarshal([]byte(req.Data), &reqData); err != nil {
		// same call-id is already exsit
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", req.Data, err)
		return simpleResponse(400), nil
	}
	log := logrus.WithFields(
		logrus.Fields{
			"user":   reqData.UserID,
			"number": reqData.Number,
		},
	)

	numb, err := h.numberHandler.CreateNumber(reqData.UserID, reqData.Number)
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
	number, err := h.numberHandler.ReleaseNumber(ctx, id)
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
	number, err := h.numberHandler.GetNumber(ctx, id)
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
			"number": id,
			"flow":   reqData.FlowID,
		},
	)
	log.Debugf("Executing processV1NumbersIDPut. number: %s", id)

	// create update number info
	tmpNumber := &number.Number{
		ID:     id,
		FlowID: reqData.FlowID,
	}
	ctx := context.Background()
	number, err := h.numberHandler.UpdateNumber(ctx, tmpNumber)
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
	number, err := h.numberHandler.GetNumberByNumber(ctx, num)
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
	tmpUserID, _ := strconv.Atoi(u.Query().Get("user_id"))
	userID := uint64(tmpUserID)

	log := logrus.WithFields(logrus.Fields{
		"user":  userID,
		"size":  pageSize,
		"token": pageToken,
	})

	ctx := context.Background()
	numbers, err := h.numberHandler.GetNumbers(ctx, userID, pageSize, pageToken)
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
