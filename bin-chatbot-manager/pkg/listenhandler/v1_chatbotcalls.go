package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1ChatbotcallsGet handles GET /v1/chatbotcall request
func (h *listenHandler) processV1ChatbotcallsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ChatbotcallsGet",
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
	customerID := uuid.FromStringOrNil(u.Query().Get("customer_id"))

	// get filters
	filters := getFilters(u)

	log = log.WithFields(logrus.Fields{
		"customer_id": customerID,
		"size":        pageSize,
		"token":       pageToken,
		"filters":     filters,
	})

	tmp, err := h.chatbotcallHandler.Gets(ctx, customerID, pageSize, pageToken, filters)
	if err != nil {
		log.Debugf("Could not get conferences. err: %v", err)
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

// processV1ChatbotcallsPost handles POST /v1/chatbotcalls request
func (h *listenHandler) processV1ChatbotcallsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ChatbotcallsPost",
		"request": m,
	})

	var req request.V1DataChatbotcallsPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.chatbotcallHandler.Start(ctx, req.ChatbotID, uuid.Nil, req.ReferenceType, req.ReferenceID, req.Gender, req.Language)
	if err != nil {
		log.Errorf("Could not create chatbotcall. err: %v", err)
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

// processV1ChatbotcallsIDGet handles GET /v1/chatbotcalls/<chatbotcall-id> request
func (h *listenHandler) processV1ChatbotcallsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ChatbotcallsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.chatbotcallHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get chatbot. err: %v", err)
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

// processV1ChatbotcallsIDDelete handles DELETE /v1/chatbotcalls/<chatbotcall-id> request
func (h *listenHandler) processV1ChatbotcallsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ChatbotcallsIDDelete",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.chatbotcallHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete chatbotcall. err: %v", err)
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

// processV1ChatbotcallsIDMessagesPost handles POST /v1/chatbotcalls/<chatbotcall-id>/messages request
func (h *listenHandler) processV1ChatbotcallsIDMessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	return nil, nil
	// log := logrus.WithFields(logrus.Fields{
	// 	"handler": "processV1ChatbotcallsIDMessagesPost",
	// 	"request": m,
	// })

	// var req request.V1DataChatbotcallsIDMessagesPost
	// if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
	// 	log.Errorf("Could not unmarshal the requested data. err: %v", err)
	// 	return nil, err
	// }

	// uriItems := strings.Split(m.URI, "/")
	// if len(uriItems) < 4 {
	// 	log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
	// 	return simpleResponse(400), nil
	// }
	// id := uuid.FromStringOrNil(uriItems[3])

	// tmp, err := h.chatbotcallHandler.ChatMessageByID(ctx, id, chatbotcall.MessageRole(req.Role), req.Text)
	// if err != nil {
	// 	log.Errorf("Could not create chatbotcall message. err: %v", err)
	// 	return simpleResponse(500), nil
	// }

	// data, err := json.Marshal(tmp)
	// if err != nil {
	// 	log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
	// 	return simpleResponse(500), nil
	// }

	// res := &sock.Response{
	// 	StatusCode: 200,
	// 	DataType:   "application/json",
	// 	Data:       data,
	// }

	// return res, nil
}
