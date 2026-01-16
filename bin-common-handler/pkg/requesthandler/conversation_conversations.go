package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvrequest "monorepo/bin-conversation-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ConversationV1ConversationGet gets the conversation
func (r *requestHandler) ConversationV1ConversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error) {

	uri := fmt.Sprintf("/v1/conversations/%s", conversationID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/conversations", requestTimeoutDefault, 0, ContentTypeNone, nil)
	if err != nil {
		return nil, err
	}

	var res cvconversation.Conversation
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1ConversationList sends a request to conversation-manager
// to getting a list of conversation info.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationList(ctx context.Context, pageToken string, pageSize uint64, filters map[cvconversation.Field]any) ([]cvconversation.Conversation, error) {
	uri := fmt.Sprintf("/v1/conversations?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/conversations", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []cvconversation.Conversation
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ConversationV1ConversationCreate sends a request to conversation-manager
// to create a conversation.
// it returns created conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	conversationType cvconversation.Type,
	dialogID string,
	self address.Address,
	peer address.Address,
) (*cvconversation.Conversation, error) {
	uri := "/v1/conversations"

	data := &cvrequest.V1DataConversationsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,
		Type:       conversationType,
		DialogID:   dialogID,
		Self:       self,
		Peer:       peer,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPost, "conversation/conversations", 3000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cvconversation.Conversation
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ConversationV1ConversationUpdate sends a request to conversation-manager
// to update the conversation info.
// it returns updated conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationUpdate(ctx context.Context, conversationID uuid.UUID, fields map[cvconversation.Field]any) (*cvconversation.Conversation, error) {
	uri := fmt.Sprintf("/v1/conversations/%s", conversationID)

	m, err := json.Marshal(fields)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPut, "conversation/conversations/<conversation_id>", 30000, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res cvconversation.Conversation
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
