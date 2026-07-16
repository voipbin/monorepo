package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/pkg/listenhandler/models/request"
)

// processV1MessagesPost handles Post /v1/messages request
func (h *listenHandler) processV1MessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesPost",
		"request": m,
	})

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.messageHandler.Create(
		ctx,
		req.CustomerID,
		req.SessionID,
		req.Direction,
		req.SenderID,
		req.Text,
	)
	if err != nil {
		log.Errorf("Could not create the message. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesGet handles Get /v1/messages request
func (h *listenHandler) processV1MessagesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	filters, err := utilhandler.ConvertFilters[message.FieldStruct, message.Field](message.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.messageHandler.List(ctx, pageSize, pageToken, filters)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDGet handles Get /v1/messages/<message-id> request
func (h *listenHandler) processV1MessagesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid message ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.messageHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get message info. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDDelete handles Delete /v1/messages/<message-id> request
func (h *listenHandler) processV1MessagesIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	if id == uuid.Nil {
		log.Errorf("Invalid message ID.")
		return simpleResponse(400), nil
	}

	tmp, err := h.messageHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete message info. err: %v", err)
		return errorResponse(err), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
