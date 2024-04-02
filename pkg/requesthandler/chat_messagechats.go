package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	chatmedia "gitlab.com/voipbin/bin-manager/chat-manager.git/models/media"
	chatmessagechat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
	chatrequest "gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/listenhandler/models/request"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ChatV1MessagechatCreate sends a request to chat-manager
// to creating a messagechat.
// it returns created chat if it succeed.
func (r *requestHandler) ChatV1MessagechatCreate(
	ctx context.Context,
	customerID uuid.UUID,
	chatID uuid.UUID,
	source commonaddress.Address,
	messageType chatmessagechat.Type,
	text string,
	medias []chatmedia.Media,
) (*chatmessagechat.Messagechat, error) {
	uri := "/v1/messagechats"

	data := &chatrequest.V1DataMessagechatsPost{
		CustomerID:  customerID,
		ChatID:      chatID,
		Source:      source,
		MessageType: messageType,
		Text:        text,
		Medias:      medias,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceChatMessagechats, requestTimeoutDefault, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatmessagechat.Messagechat
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatV1MessagechatGet sends a request to chat-manager
// to getting a messagechat.
// it returns given messagechat id's chat if it succeed.
func (r *requestHandler) ChatV1MessagechatGet(ctx context.Context, messagechatID uuid.UUID) (*chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats/%s", messagechatID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatMessagechats, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatmessagechat.Messagechat
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ChatV1MessagechatGets sends a request to chat-manager
// to getting a list of messagechat info.
// it returns detail list of chat info if it succeed.
func (r *requestHandler) ChatV1MessagechatGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = parseFilters(uri, filters)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceChatMessagechats, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []chatmessagechat.Messagechat
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ChatV1MessagechatDelete sends a request to chat-manager
// to delete the messagechat.
// it returns error if it went wrong.
func (r *requestHandler) ChatV1MessagechatDelete(ctx context.Context, chatID uuid.UUID) (*chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats/%s", chatID)

	tmp, err := r.sendRequestChat(ctx, uri, rabbitmqhandler.RequestMethodDelete, resourceChatMessagechats, requestTimeoutDefault, 0, ContentTypeJSON, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res chatmessagechat.Messagechat
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
