package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	cbrequest "monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

// ChatbotV1ServiceTypeChabotcallStart sends a request to chat-manager
// to starts a chatbotcall service.
// it returns created service if it succeed.
func (r *requestHandler) ChatbotV1ServiceTypeChabotcallStart(
	ctx context.Context,
	chatbotID uuid.UUID,
	activeflowID uuid.UUID,
	referenceType cbchatbotcall.ReferenceType,
	referenceID uuid.UUID,
	gender cbchatbotcall.Gender,
	language string,
	requestTimeout int,
) (*service.Service, error) {
	uri := "/v1/services/type/chatbotcall"

	data := &cbrequest.V1DataServicesTypeChatbotcallPost{
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

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodPost, "chatbot/services/type/chatbotcall", requestTimeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res service.Service
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
