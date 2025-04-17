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
)

// ConversationV1ConversationGet gets the conversation
func (r *requestHandler) ConversationV1ConversationGet(ctx context.Context, conversationID uuid.UUID) (*cvconversation.Conversation, error) {

	uri := fmt.Sprintf("/v1/conversations/%s", conversationID)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/conversations", requestTimeoutDefault, 0, ContentTypeNone, nil)
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

// ConversationV1ConversationGets sends a request to conversation-manager
// to getting a list of conversation info.
// it returns detail list of conversation info if it succeed.
func (r *requestHandler) ConversationV1ConversationGets(ctx context.Context, pageToken string, pageSize uint64, filters map[string]string) ([]cvconversation.Conversation, error) {
	uri := fmt.Sprintf("/v1/conversations?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	// parse filters
	uri = r.utilHandler.URLMergeFilters(uri, filters)

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodGet, "conversation/conversations", 30000, 0, ContentTypeNone, nil)
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

	tmp, err := r.sendRequestConversation(ctx, uri, sock.RequestMethodPut, "conversation/conversations/<conversation_id>", 30000, 0, ContentTypeJSON, m)
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
