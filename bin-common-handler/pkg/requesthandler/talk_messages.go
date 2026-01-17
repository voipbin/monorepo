package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	talkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TalkV1MessageGet gets a message by ID
func (r *requestHandler) TalkV1MessageGet(ctx context.Context, messageID uuid.UUID) (*talkmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", messageID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/messages", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get message: status %d", res.StatusCode)
	}

	var message talkmessage.Message
	if err := json.Unmarshal(res.Data, &message); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal message")
	}

	return &message, nil
}

// TalkV1MessageCreate creates a new message
func (r *requestHandler) TalkV1MessageCreate(
	ctx context.Context,
	chatID uuid.UUID,
	parentID *uuid.UUID,
	ownerType string,
	ownerID uuid.UUID,
	msgType talkmessage.Type,
	text string,
) (*talkmessage.Message, error) {
	uri := "/v1/messages"

	data := map[string]any{
		"chat_id":    chatID.String(),
		"owner_type": ownerType,
		"owner_id":   ownerID.String(),
		"type":       string(msgType),
		"text":       text,
	}

	if parentID != nil {
		data["parent_id"] = parentID.String()
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/messages", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create message: status %d", res.StatusCode)
	}

	var message talkmessage.Message
	if err := json.Unmarshal(res.Data, &message); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal message")
	}

	return &message, nil
}

// TalkV1MessageDelete deletes a message (soft delete)
func (r *requestHandler) TalkV1MessageDelete(ctx context.Context, messageID uuid.UUID) (*talkmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s", messageID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/messages", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete message: status %d", res.StatusCode)
	}

	var message talkmessage.Message
	if err := json.Unmarshal(res.Data, &message); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal message")
	}

	return &message, nil
}

// TalkV1MessageList gets a list of messages (simplified - for future expansion)
func (r *requestHandler) TalkV1MessageList(ctx context.Context, pageToken string, pageSize uint64) ([]*talkmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages?page_token=%s&page_size=%d", pageToken, pageSize)

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/messages", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list messages: status %d", res.StatusCode)
	}

	var messages []*talkmessage.Message
	if err := json.Unmarshal(res.Data, &messages); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal messages")
	}

	return messages, nil
}

// TalkV1MessageReactionCreate adds a reaction to a message
func (r *requestHandler) TalkV1MessageReactionCreate(
	ctx context.Context,
	messageID uuid.UUID,
	ownerType string,
	ownerID uuid.UUID,
	emoji string,
) (*talkmessage.Message, error) {
	uri := fmt.Sprintf("/v1/messages/%s/reactions", messageID.String())

	data := map[string]any{
		"owner_type": ownerType,
		"owner_id":   ownerID.String(),
		"emoji":      emoji,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/reactions", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create reaction: status %d", res.StatusCode)
	}

	var message talkmessage.Message
	if err := json.Unmarshal(res.Data, &message); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal message")
	}

	return &message, nil
}
