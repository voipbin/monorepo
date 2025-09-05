package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	chatmedia "monorepo/bin-chat-manager/models/media"
	chatmessagechat "monorepo/bin-chat-manager/models/messagechat"
	chatrequest "monorepo/bin-chat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
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

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodPost, "chat/messagechats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res chatmessagechat.Messagechat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1MessagechatGet sends a request to chat-manager
// to getting a messagechat.
// it returns given messagechat id's chat if it succeed.
func (r *requestHandler) ChatV1MessagechatGet(ctx context.Context, messagechatID uuid.UUID) (*chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats/%s", messagechatID)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodGet, "chat/messagechats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res chatmessagechat.Messagechat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1MessagechatGets sends a request to chat-manager
// to getting a list of messagechat info.
// it returns detail list of chat info if it succeed.
func (r *requestHandler) ChatV1MessagechatGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodGet, "chat/messagechats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res []chatmessagechat.Messagechat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ChatV1MessagechatDelete sends a request to chat-manager
// to delete the messagechat.
// it returns error if it went wrong.
func (r *requestHandler) ChatV1MessagechatDelete(ctx context.Context, chatID uuid.UUID) (*chatmessagechat.Messagechat, error) {
	uri := fmt.Sprintf("/v1/messagechats/%s", chatID)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodDelete, "chat/messagechats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res chatmessagechat.Messagechat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
