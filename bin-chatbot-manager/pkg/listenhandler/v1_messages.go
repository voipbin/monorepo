package listenhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"net/url"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1MessagesGet handles GET /v1/messages request
func (h *listenHandler) processV1MessagesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
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

	// get customer id
	chatbotcallID := uuid.FromStringOrNil(u.Query().Get("chatbotcall_id"))

	// get filters
	filters := getFilters(u)

	log = log.WithFields(logrus.Fields{
		"customer_id": chatbotcallID,
		"size":        pageSize,
		"token":       pageToken,
		"filters":     filters,
	})

	tmp, err := h.messageHandler.Gets(ctx, chatbotcallID, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get messages. err: %v", err)
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

// processV1MessagesPost handles POST /v1/messages request
func (h *listenHandler) processV1MessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1MessagesPost",
		"request": m,
	})

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.messageHandler.Send(ctx, req.ChatbotcallID, req.Role, req.Content)
	if err != nil {
		log.Errorf("Could not create chatbot. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDGet handles POST /v1/messages/<message-id> request
func (h *listenHandler) processV1MessagesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1MessagesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.messageHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not create chatbot. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
