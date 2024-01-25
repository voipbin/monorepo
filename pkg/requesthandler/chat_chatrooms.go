package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ChatV1ChatGet sends a request to chat-manager
// to getting a chat.
// it returns given chat id's chat if it succeed.
func (r *requestHandler) ChatV1ChatroomGet(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error) {
	uri := fmt.Sprintf("/v1/chatrooms/%s", chatroomID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatChatrooms, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatchatroom.Chatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatV1ChatroomGets sends a request to chat-manager
// to getting a list of chatroom info.
// it returns detail list of chatroom info if it succeed.
func (r *requestHandler) ChatV1ChatroomGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatchatroom.Chatroom, error) {
	uri := fmt.Sprintf("/v1/chatrooms?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	for k, v := range filters {
		uri = fmt.Sprintf("%s&filter_%s=%s", uri, k, v)
	}

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatChatrooms, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []chatchatroom.Chatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatV1ChatroomDelete sends a request to chat-manager
// to delete the chatroom.
// it returns error if it went wrong.
func (r *requestHandler) ChatV1ChatroomDelete(ctx context.Context, chatroomID uuid.UUID) (*chatchatroom.Chatroom, error) {
	uri := fmt.Sprintf("/v1/chatrooms/%s", chatroomID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceChatChatrooms, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatchatroom.Chatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
