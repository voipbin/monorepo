package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatrequest "monorepo/bin-chat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

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

// ChatV1ChatroomUpdateBasicInfo sends a request to chat-manager
// to update the chatroom's basic info.
func (r *requestHandler) ChatV1ChatroomUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchatroom.Chatroom, error) {
	uri := fmt.Sprintf("/v1/chatrooms/%s", id)

	data := &chatrequest.V1DataChatroomsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodPut, resourceChatChats, requestTimeoutDefault, 0, ContentTypeJSON, m)
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
