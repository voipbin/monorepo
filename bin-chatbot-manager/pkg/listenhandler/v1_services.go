package listenhandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"
)

// processV1ServicesTypeChatbotcallPost handles POST /v1/services/type/chatbotcall request
func (h *listenHandler) processV1ServicesTypeChatbotcallPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1ServicesTypeChatbotcallPost",
		"request": m,
	})

	var req request.V1DataServicesTypeChatbotcallPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return nil, err
	}

	tmp, err := h.chatbotcallHandler.ServiceStart(ctx, req.CustomerID, req.ChatbotID, req.ActiveflowID, req.ReferenceType, req.ReferenceID, req.Gender, req.Language)
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
