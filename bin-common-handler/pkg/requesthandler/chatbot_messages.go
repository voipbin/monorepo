package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	cbmessage "monorepo/bin-chatbot-manager/models/message"
	cbrequest "monorepo/bin-chatbot-manager/pkg/listenhandler/models/request"

	"monorepo/bin-common-handler/models/sock"
	"net/url"

	"github.com/gofrs/uuid"
)

// ChatbotV1MessageGetsByChatbotcallID sends a request to chatbot-manager
// to getting a list of messages info of the given chatbotcall id.
// it returns detail list of message info if it succeed.
func (r *requestHandler) ChatbotV1MessageGetsByChatbotcallID(ctx context.Context, chatbotcallID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d&chatbotcall_id=%s", url.QueryEscape(pageToken), pageSize, chatbotcallID)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodGet, "chatbot/messages", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cbmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatbotV1MessageSend sends a request to chatbot-manager
// to send a message.
// it returns created message if it succeed.
func (r *requestHandler) ChatbotV1MessageSend(ctx context.Context, chatbotcallID uuid.UUID, role cbmessage.Role, content string, timeout int) (*cbmessage.Message, error) {
	uri := "/v1/messages"

	data := &cbrequest.V1DataMessagesPost{
		ChatbotcallID: chatbotcallID,
		Role:          role,
		Content:       content,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodPost, "chatbot/messages", timeout, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotV1MessageGet returns the message.
func (r *requestHandler) ChatbotV1MessageGet(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", messageID.String())

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodGet, "chatbot/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not get conference. status: %d", tmp.StatusCode)
	}

	var res cbmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatbotV1MessageDelete sends a request to chatbot-manager
// to deleting a message.
// it returns deleted message if it succeed.
func (r *requestHandler) ChatbotV1MessageDelete(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", messageID)

	tmp, err := r.sendRequestChatbot(ctx, uri, sock.RequestMethodDelete, "chatbot/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cbmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
