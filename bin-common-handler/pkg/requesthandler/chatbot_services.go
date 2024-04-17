package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	cbservice "monorepo/bin-chatbot-manager/models/service"
	cbrequest "monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// ChatbotV1ServiceTypeChabotcallStart sends a request to chat-manager
// to starts a chatbotcall service.
// it returns created service if it succeed.
func (r *requestHandler) ChatbotV1ServiceTypeChabotcallStart(
	ctx context.Context,
	customerID uuid.UUID,
	chatbotID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType cbchatbotcall.ReferenceType,
	referenceID uuid.UUID,
	gender cbchatbotcall.Gender,
	language string,
	requestTimeout int,
) (*cbservice.Service, error) {
	uri := "/v1/services/type/chatbotcall"

	data := &cbrequest.V1DataServicesTypeChatbotcallPost{
		CustomerID:    customerID,
		ChatbotID:     chatbotID,
		ActiveflowID:  activeflowID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
		Gender:        gender,
		Language:      language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceChatbotServiceTypeChatbotcall, requestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbservice.Service
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
