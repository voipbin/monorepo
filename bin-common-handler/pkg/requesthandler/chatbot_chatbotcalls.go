package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// ChatbotV1ChatbotcallGetsByCustomerID sends a request to chatbot-manager
// to getting a list of chatbotcall info of the given customer id.
// it returns detail list of chatbotcall info if it succeed.
func (r *requestHandler) ChatbotV1ChatbotcallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbchatbotcall.Chatbotcall, error) {
	uri := fmt.Sprintf("/v1/chatbotcalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodGet, "chatbot/chatbotcalls", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cbchatbotcall.Chatbotcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatbotV1ChatbotcallGet returns the chatbot.
func (r *requestHandler) ChatbotV1ChatbotcallGet(ctx context.Context, chatbotcallID uuid.UUID) (*cbchatbotcall.Chatbotcall, error) {

	uri := fmt.Sprintf("/v1/chatbotcalls/%s", chatbotcallID)

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodGet, "chatbot/chatbotcalls/<chatbotcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res cbchatbotcall.Chatbotcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotV1ChatbotcallDelete sends a request to chatbot-manager
// to deleting a chatbotcall.
// it returns deleted conference if it succeed.
func (r *requestHandler) ChatbotV1ChatbotcallDelete(ctx context.Context, chatbotcallID uuid.UUID) (*cbchatbotcall.Chatbotcall, error) {
	uri := fmt.Sprintf("/v1/chatbotcalls/%s", chatbotcallID)

	tmp, err := r.sendRequestChatbot(ctx, uri, rabbitmqhandler.RequestMethodDelete, "chatbot/chatbotcalls/<chatbotcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbchatbotcall.Chatbotcall
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
