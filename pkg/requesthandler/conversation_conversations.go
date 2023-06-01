package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/gofrs/uuid"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	cvrequest "gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/listenhandler/models/request"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// ConversationV1ConversationGet gets the conversation
func (r *requestHandler) ConversationV1ConversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error) {

	uri := fmt.Sprintf("/v1/conversations/%s", conversationID)

	tmp, err := r.sendRequestConversation(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceConversationConversations, requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	if tmp.StatusCode >= 299 {
		return nil, fmt.Errorf("could not send the message")
	}

	var res cvconversation.Conversation
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1ConversationGetsByCustomerID sends a request to conversation-manager
// to getting a list of conversation info.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationGetsByCustomerID(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64) ([]cvconversation.Conversation, error) {
	uri := fmt.Sprintf("/v1/conversations?page_token=%s&page_size=%d&customer_id=%s", url.QueryEscape(pageToken), pageSize, customerID)

	tmp, err := r.sendRequestConversation(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceConversationConversations, 30000, 0, ContentTypeNone, nil)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res []cvconversation.Conversation
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return res, nil
}

// ConversationV1ConversationUpdate sends a request to conversation-manager
// to update the conversation info.
// it returns updated conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationUpdate(ctx context.Context, conversationID uuid.UUID, name string, detail string) (*cvconversation.Conversation, error) {
	uri := fmt.Sprintf("/v1/conversations/%s", conversationID)

	data := &cvrequest.V1DataConversationsIDPut{
		Name:   name,
		Detail: detail,
	}
	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, rabbitmqhandler.RequestMethodPut, "conversation/conversations/<conversation_id>", 30000, 0, ContentTypeJSON, m)
	switch {
	case err != nil:
		return nil, err
	case tmp == nil:
		// not found
		return nil, fmt.Errorf("response code: %d", 404)
	case tmp.StatusCode > 299:
		return nil, fmt.Errorf("response code: %d", tmp.StatusCode)
	}

	var res cvconversation.Conversation
	if err := json.Unmarshal([]byte(tmp.Data), &res); err != nil {
		return nil, err
	}

	return &res, nil
}

// ConversationV1MessageSend sends a request to conversation-manager
// to send a message to the given conversation.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1MessageSend(ctx context.Context, conversationID uuid.UUID, text string, medias []cvmedia.Media) (*cvmessage.Message, error) {
	uri := fmt.Sprintf("/v1/conversations/%s/messages", conversationID)

	req := &cvrequest.V1DataConversationsIDMessagesPost{
		Text:   text,
		Medias: medias,
	}

	m, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceConversationConversationsIDMessages, 30000, 0, ContentTypeJSON, m)
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

// ConversationV1ConversationMessageGetsByConversationID sends a request to conversation-manager
// to getting a list of conversation's messages.
// it returns detail list of messages info if it succeed.
func (r *requestHandler) ConversationV1ConversationMessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, pageToken string, pageSize uint64) ([]cvmessage.Message, error) {
	uri := fmt.Sprintf("/v1/conversations/%s/messages?page_token=%s&page_size=%d", conversationID, url.QueryEscape(pageToken), pageSize)

	tmp, err := r.sendRequestConversation(ctx, uri, rabbitmqhandler.RequestMethodGet, resourceConversationConversationsIDMessages, 30000, 0, ContentTypeNone, nil)
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
