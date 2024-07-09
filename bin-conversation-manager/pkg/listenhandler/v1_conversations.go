package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
)

// processV1ConversationsGet handles
// /v1/conversations GET
func (h *listenHandler) processV1ConversationsGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConversationsGet",
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

	// get filters
	filters := h.utilHandler.URLParseFilters(u)

	tmps, err := h.conversationHandler.Gets(ctx, pageToken, pageSize, filters)
	if err != nil {
		log.Debugf("Could not get conversations. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmps)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmps, err)
		return simpleResponse(500), nil
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConversationsIDGet handles
// /v1/conversations/{id} GET
func (h *listenHandler) processV1ConversationsIDGet(ctx context.Context, req *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":            "processV1ConversationsIDGet",
		"conversation_id": id,
	})
	log.Debugf("Executing processV1ConversationsIDGet. message_id: %s", id)

	tmp, err := h.conversationHandler.Get(ctx, id)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
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

// processV1ConversationsIDPut handles
// /v1/conversations/{id} PUT
func (h *listenHandler) processV1ConversationsIDPut(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":            "processV1ConversationsIDPut",
		"conversation_id": id,
	})

	var req request.V1DataConversationsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}
	log.Debugf("Executing processV1ConversationsIDPut. message_id: %s", id)

	tmp, err := h.conversationHandler.Update(ctx, id, req.Name, req.Detail)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
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

// processV1ConversationsIDMessagesGet handles
// /v1/conversations/<conversation-id>/messages GET
func (h *listenHandler) processV1ConversationsIDMessagesGet(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConversationsIDMessagesGet",
		"request": m,
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get conversation id
	tmpVals := strings.Split(u.Path, "/")
	conversationID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.messageHandler.GetsByConversationID(ctx, conversationID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not get conversation message. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConversationsIDMessagesPost handles
// /v1/conversations/<conversation-id>/messages POST
func (h *listenHandler) processV1ConversationsIDMessagesPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConversationsIDMessagesPost",
		"request": m,
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// get conversation id
	tmpVals := strings.Split(u.Path, "/")
	conversationID := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataConversationsIDMessagesPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.conversationHandler.MessageSend(ctx, conversationID, req.Text, req.Medias)
	if err != nil {
		log.Errorf("Could not send the message correctly. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &rabbitmqhandler.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
