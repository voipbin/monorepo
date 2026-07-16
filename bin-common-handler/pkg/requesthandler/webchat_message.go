package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"

	wcmessage "monorepo/bin-webchat-manager/models/message"
	wcrequest "monorepo/bin-webchat-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// WebchatV1MessageCreate sends a request to webchat-manager to create a message.
func (r *requestHandler) WebchatV1MessageCreate(
	ctx context.Context,
	customerID uuid.UUID,
	sessionID uuid.UUID,
	direction wcmessage.Direction,
	senderID uuid.UUID,
	text string,
) (*wcmessage.Message, error) {
	uri := "/v1/messages"

	data := &wcrequest.V1DataMessagesPost{
		CustomerID: customerID,
		SessionID:  sessionID,
		Direction:  direction,
		SenderID:   senderID,
		Text:       text,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodPost, "webchat/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res wcmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1MessageGet sends a request to webchat-manager to get the message.
func (r *requestHandler) WebchatV1MessageGet(ctx context.Context, id uuid.UUID) (*wcmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// WebchatV1MessageList sends a request to webchat-manager to get a list of messages.
func (r *requestHandler) WebchatV1MessageList(ctx context.Context, pageToken string, pageSize uint64, filters map[wcmessage.Field]any) ([]*wcmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", pageToken, pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodGet, "webchat/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []*wcmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// WebchatV1MessageDelete sends a request to webchat-manager to delete the message.
func (r *requestHandler) WebchatV1MessageDelete(ctx context.Context, id uuid.UUID) (*wcmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", id)

	tmp, err := r.sendRequestWebchat(ctx, uri, sock.RequestMethodDelete, "webchat/messages/<message-id>", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res wcmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
