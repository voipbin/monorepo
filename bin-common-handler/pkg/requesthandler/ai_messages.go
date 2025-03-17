package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	cbmessage "monorepo/bin-ai-manager/models/message"
	cbrequest "monorepo/bin-ai-manager/pkg/listenhandler/models/request"

	"monorepo/bin-common-handler/models/sock"
	"net/url"

	"github.com/gofrs/uuid"
)

// AIV1MessageGetsByAIcallID sends a request to ai-manager
// to getting a list of messages info of the given chatbotcall id.
// it returns detail list of message info if it succeed.
func (r *requestHandler) AIV1MessageGetsByAIcallID(ctx context.Context, chatbotcallID uuid.UUID, pageToken string, pageSize uint64, filters map[string]string) ([]cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d&chatbotcall_id=%s", url.QueryEscape(pageToken), pageSize, chatbotcallID)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "chatbot/messages", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// AIV1MessageSend sends a request to ai-manager
// to send a message.
// it returns created message if it succeed.
func (r *requestHandler) AIV1MessageSend(ctx context.Context, chatbotcallID uuid.UUID, role cbmessage.Role, content string, timeout int) (*cbmessage.Message, error) {
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

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodPost, "chatbot/messages", timeout, 0, ContentTypeJSON, m)
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

// AIV1MessageGet returns the message.
func (r *requestHandler) AIV1MessageGet(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", messageID.String())

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodGet, "chatbot/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// AIV1MessageDelete sends a request to ai-manager
// to deleting a message.
// it returns deleted message if it succeed.
func (r *requestHandler) AIV1MessageDelete(ctx context.Context, messageID uuid.UUID) (*cbmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", messageID)

	tmp, err := r.sendRequestAI(ctx, uri, sock.RequestMethodDelete, "chatbot/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
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
