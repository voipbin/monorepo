package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/listenhandler/models/request"
)

// processV1ConversationsGet handles
// /v1/conversations GET
func (h *listenHandler) processV1ConversationsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	var req map[string]any
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"raw_filters": req,
	}).Debug("Received filters from API request (check customer_id value type)")

	fields, err := conversation.ConvertStringMapToFieldMap(req)
	if err != nil {
		log.Errorf("Could not convert the filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log.WithFields(logrus.Fields{
		"converted_filters": fields,
	}).Debug("Filters after conversion (customer_id should be uuid.UUID)")

	tmps, err := h.conversationHandler.Gets(ctx, pageToken, pageSize, fields)
	if err != nil {
		log.Debugf("Could not get conversations. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmps)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmps, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConversationsPost handles
// POST /v1/conversations request
func (h *listenHandler) processV1ConversationsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConversationsPost",
		"request": m,
	})

	var req request.V1DataConversationsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Debugf("Could not unmarshal the data. data: %v, err: %v", m.Data, err)
		return simpleResponse(400), nil
	}

	tmp, err := h.conversationHandler.Create(
		ctx,
		req.CustomerID,
		req.Name,
		req.Detail,
		conversation.Type(req.Type),
		req.DialogID,
		req.Self,
		req.Peer,
	)
	if err != nil {
		log.Errorf("Could not create the account. err: %v", err)
		return simpleResponse(500), nil
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

// processV1ConversationsIDGet handles
// /v1/conversations/{id} GET
func (h *listenHandler) processV1ConversationsIDGet(ctx context.Context, req *sock.Request) (*sock.Response, error) {
	uriItems := strings.Split(req.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log := logrus.WithFields(logrus.Fields{
		"func":            "processV1ConversationsIDGet",
		"conversation_id": id,
	})
	log.Debugf("Executing processV1ConversationsIDGet. conversation_id: %s", id)

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

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1ConversationsIDPut handles
// /v1/conversations/{id} PUT
func (h *listenHandler) processV1ConversationsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ConversationsIDPut",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), nil
	}

	id := uuid.FromStringOrNil(uriItems[3])
	log.Debugf("Executing processV1ConversationsIDPut. message_id: %s", id)

	allowedItems := []string{
		string(conversation.FieldOwnerType),
		string(conversation.FieldOwnerID),
		string(conversation.FieldName),
		string(conversation.FieldDetail),
		string(conversation.FieldAccountID),
	}

	filteredItems, err := requesthandler.GetFilteredItems(m, allowedItems)
	if err != nil {
		log.Errorf("Could not filter the request. err: %v", err)
		return nil, err
	}
	if len(filteredItems) == 0 {
		log.Debugf("No allowed fields provided for update. Skipping.")
		return simpleResponse(200), nil
	}

	tmpFields, err := conversation.ConvertStringMapToFieldMap(filteredItems)
	if err != nil {
		log.Errorf("Could not convert field map. err: %v", err)
		return nil, err
	}

	tmp, err := h.conversationHandler.Update(ctx, id, tmpFields)
	if err != nil {
		log.Debugf("Could not get a conversation. conversation_id: %s, err: %v", id, err)
		return simpleResponse(500), nil
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
