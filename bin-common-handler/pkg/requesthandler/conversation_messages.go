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
)

// ConversationV1MessageGet gets the message
func (r *requestHandler) ConversationV1MessageGet(ctx context.Context, messageID uuid.UUID) (*cvmessage.Message, error) {

	uri := fmt.Sprintf("/v1/messages/%s", messageID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/messages", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res cvmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1MessageGets sends a request to conversation-manager
// to getting a list of message info.
// it returns detail list of message info if it succeed.
func (r *requestHandler) ConversationV1MessageGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cvmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/messages", 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cvmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ConversationV1MessageCreate sends a request to conversation-manager
// to create a message. This does not send the message to the conversation.
// it returns created message info if it succeed.
func (r *requestHandler) ConversationV1MessageCreate(
	ctx context.Context,
	customerID uuid.UUID,
	conversationID uuid.UUID,
	direction cvmessage.Direction,
	status cvmessage.Status,
	referenceType cvmessage.ReferenceType,
	referenceID string,
	transactionID string,
	text string,
	medias []cvmedia.Media,
) (*cvmessage.Message, error) {
	uri := "/v1/messages/create"

	data := &cvrequest.V1DataMessagesCreatePost{
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
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cvmessage.Message
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}
