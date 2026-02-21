package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/sock"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	cvrequest "monorepo/bin-conversation-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ConversationV1MessageGet gets the message
func (r *requestHandler) ConversationV1MessageGet(ctx context.Context, messageID uuid.UUID) (*cvmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", messageID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/messages", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cvmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1MessageSend sends a request to conversation-manager
// to send a message to the given conversation.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []cvmedia.Media) (*cvmessage.Message, error) {
	uri := "/v1/messages"

	req := &cvrequest.V1DataMessagesPost{
		ConversationID: conversationID,
		Text:           text,
		Medias:         medias,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "conversation/messages", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cvmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1MessageList sends a request to conversation-manager
// to getting a list of message info.
// it returns detail list of message info if it succeed.
func (r *requestHandler) ConversationV1MessageList(ctx context.Context, pageToken string, pageSize uint64, filters map[cvmessage.Field]any) ([]cvmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/messages", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cvmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ConversationV1MessageCreate sends a request to conversation-manager
// to create a message. This does not send the message to the conversation.
// it returns created message info if it succeed.
func (r *requestHandler) ConversationV1MessageCreate(
	ctx context.Context,
	id uuid.UUID,
	customerID uuid.UUID,
	conversationID uuid.UUID,
	direction cvmessage.Direction,
	status cvmessage.Status,
	referenceType cvmessage.ReferenceType,
	referenceID uuid.UUID,
	transactionID string,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.Message, error) {
	uri := "/v1/messages/create"

	data := &cvrequest.V1DataMessagesCreatePost{
		ID:             id,
		CustomerID:     customerID,
		ConversationID: conversationID,
		Direction:      direction,
		Status:         status,
		ReferenceType:  referenceType,
		ReferenceID:    referenceID,
		TransactionID:  transactionID,
		Text:           text,
		Medias:         medias,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "conversation/messages", 3000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cvmessage.Message
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
