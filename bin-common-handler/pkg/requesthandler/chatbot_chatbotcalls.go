package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
	cbrequest "monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
)

func (r *requestHandler) ChatbotV1ChatbotcallStart(ctx context.Context, chatbotID uuid.UUID, referenceType cbchatbotcall.ReferenceType, referenceID uuid.UUID, gender cbchatbotcall.Gender, language string) (*cbchatbotcall.Chatbotcall, error) {
	uri := "/v1/chatbotcalls"

	data := &cbrequest.V1DataChatbotcallsPost{
		ChatbotID: chatbotID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Gender:   gender,
		Language: language,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodPost, "chatbot/chatbotcalls", requestTimeoutDefault, 0, ContentTypeJSON, m)
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

// ChatbotV1ChatbotcallGetsByCustomerID sends a request to chatbot-manager
// to getting a list of chatbotcall info of the given customer id.
// it returns detail list of chatbotcall info if it succeed.
func (r *requestHandler) ChatbotV1ChatbotcallGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbchatbotcall.Chatbotcall, error) {
	uri := fmt.Sprintf("/v1/chatbotcalls?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodGet, "chatbot/chatbotcalls", 30000, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodGet, "chatbot/chatbotcalls/<chatbotcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodDelete, "chatbot/chatbotcalls/<chatbotcall-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
