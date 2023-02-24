package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cbchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	cbrequest "gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ChatbotV1ChatbotGetsByCustomerID sends a request to chatbot-manager
// to getting a list of chatbot info of the given customer id.
// it returns detail list of chatbot info if it succeed.
func (r *requestHandler) ChatbotV1ChatbotGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cbchatbot.Chatbot, error) {
	uri := fmt.Sprintf("/v1/chatbots?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatbotChatbots, 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cbchatbot.Chatbot
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatbotV1ChatbotGet returns the chatbot.
func (r *requestHandler) ChatbotV1ChatbotGet(ctx context.Context, chatbotID uuid.UUID) (*cbchatbot.Chatbot, error) {

	uri := fmt.Sprintf("/v1/chatbots/%s", chatbotID.String())

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatbotChatbotsID, requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res cbchatbot.Chatbot
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotV1ChatbotCreate sends a request to chat-manager
// to creating a chatbot.
// it returns created chat if it succeed.
func (r *requestHandler) ChatbotV1ChatbotCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType cbchatbot.EngineType,
) (*cbchatbot.Chatbot, error) {
	uri := "/v1/chatbots"

	data := &cbrequest.V1DataChatbotsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		EngineType: engineType,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceChatbotChatbots, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbchatbot.Chatbot
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotV1ChatbotDelete sends a request to chatbot-manager
// to deleting a chatbot.
// it returns deleted conference if it succeed.
func (r *requestHandler) ChatbotV1ChatbotDelete(ctx context.Context, chatbotID uuid.UUID) (*cbchatbot.Chatbot, error) {
	uri := fmt.Sprintf("/v1/chatbots/%s", chatbotID)

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceChatbotChatbotsID, requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbchatbot.Chatbot
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
