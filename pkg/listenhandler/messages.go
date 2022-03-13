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

	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/listenhandler/models/request"
)

// processV1MessagesPost handles POST /v1/messages request
func (h *listenHandler) processV1MessagesGet(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	u, err := url.Parse(m.URI)
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

	messages, err := h.messageHandler.Gets(ctx, customerID, pageSize, pageToken)
	if err != nil {
		log.Debugf("Could not get messages. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(messages)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", messages, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesPost handles POST /v1/messages request
func (h *listenHandler) processV1MessagesPost(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	ctx := context.Background()

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}
	log := logrus.WithFields(
		logrus.Fields{
			"func": "processV1MessagesPost",
			"user": req.CustomerID,
		},
	)

	// send message
	ms, err := h.messageHandler.Send(ctx, req.CustomerID, req.Source, req.Destinations, req.Text)
	if err != nil {
		log.Errorf("Could not handle the order number. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(ms)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", ms, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDGet handles GET /v1/messages/<id> request
func (h *listenHandler) processV1MessagesIDGet(req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(
		logrus.Fields{
			"id": id,
		})
	log.Debugf("Executing processV1MessagesIDGet. number: %s", id)

	ctx := context.Background()
	number, err := h.messageHandler.Get(ctx, id)
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
