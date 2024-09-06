package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-message-manager/pkg/listenhandler/models/request"
)

// processV1MessagesGet handles GET /v1/messages request
func (h *listenHandler) processV1MessagesGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesGet",
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

	// get user_id
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

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
func (h *listenHandler) processV1MessagesPost(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesPost",
		"request": m,
	})

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		logrus.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	// send message
	ms, err := h.messageHandler.Send(ctx, req.ID, req.CustomerID, req.Source, req.Destinations, req.Text)
	if err != nil {
		log.Errorf("Could not send a message. err: %v", err)
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
func (h *listenHandler) processV1MessagesIDGet(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.messageHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a message. message_id: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDDelete handles DELETE /v1/messages/<id> request
func (h *listenHandler) processV1MessagesIDDelete(ctx context.Context, m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.messageHandler.Delete(ctx, id)
	if err != nil {
		log.Debugf("Could not get a message. message_id: %s, err: %v", id, err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
