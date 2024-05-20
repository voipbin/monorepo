package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

// ChatV1MessagechatroomGets sends a request to chat-manager
// to getting a list of messagechatroom info of the given chatroom id.
// it returns detail list of messagechatroom info if it succeed.
func (r *requestHandler) ChatV1MessagechatroomGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatmessagechatroom.Messagechatroom, error) {
	uri := fmt.Sprintf("/v1/messagechatrooms?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, "chat/messagechatrooms", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []chatmessagechatroom.Messagechatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatV1MessagechatroomGet sends a request to chat-manager
// to getting a messagechatroom.
// it returns given messagechatroom id's chat if it succeed.
func (r *requestHandler) ChatV1MessagechatroomGet(ctx context.Context, messagechatroomID uuid.UUID) (*chatmessagechatroom.Messagechatroom, error) {
	uri := fmt.Sprintf("/v1/messagechatrooms/%s", messagechatroomID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, "chat/messagechatrooms", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatmessagechatroom.Messagechatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatV1MessagechatroomDelete sends a request to chat-manager
// to delete the messagechatroom.
// it returns error if it went wrong.
func (r *requestHandler) ChatV1MessagechatroomDelete(ctx context.Context, messagechatroomID uuid.UUID) (*chatmessagechatroom.Messagechatroom, error) {
	uri := fmt.Sprintf("/v1/messagechatrooms/%s", messagechatroomID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodDelete, "chat/messagechatrooms", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatmessagechatroom.Messagechatroom
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
