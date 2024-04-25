package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cbchatbot "monorepo/bin-chatbot-manager/models/chatbot"
	cbrequest "monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// ChatbotV1ChatbotGetsByCustomerID sends a request to chatbot-manager
// to getting a list of chatbot info of the given customer id.
// it returns detail list of chatbot info if it succeed.
func (r *requestHandler) ChatbotV1ChatbotGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbchatbot.Chatbot, error) {
	uri := fmt.Sprintf("/v1/chatbots?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

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

// ChatbotV1ChatbotCreate sends a request to chatbot-manager
// to creating a chatbot.
// it returns created chat if it succeed.
func (r *requestHandler) ChatbotV1ChatbotCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineType cbchatbot.EngineType,
	initPrompt string,
) (*cbchatbot.Chatbot, error) {
	uri := "/v1/chatbots"

	data := &cbrequest.V1DataChatbotsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		EngineType: engineType,
		InitPrompt: initPrompt,
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

// ChatbotV1ChatbotUpdate sends a request to chatbot-manager
// to updating a chatbot.
// it returns updated chatbot if it succeed.
func (r *requestHandler) ChatbotV1ChatbotUpdate(
	ctx context.Context,
	chatbotID uuid.UUID,
	name string,
	detail string,
	engineType cbchatbot.EngineType,
	initPrompt string,
) (*cbchatbot.Chatbot, error) {
	uri := fmt.Sprintf("/v1/chatbots/%s", chatbotID)

	data := &cbrequest.V1DataChatbotsIDPut{
		Name:       name,
		Detail:     detail,
		EngineType: engineType,
		InitPrompt: initPrompt,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceChatbotChatbotsID, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
