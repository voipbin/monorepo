package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	tkchat "monorepo/bin-talk-manager/models/chat"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TalkV1ChatGet gets a talk by ID
func (r *requestHandler) TalkV1ChatGet(ctx context.Context, talkID uuid.UUID) (*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", talkID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get talk: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &chat, nil
}

// TalkV1ChatCreate creates a new talk
func (r *requestHandler) TalkV1ChatCreate(ctx context.Context, customerID uuid.UUID, talkType tkchat.Type) (*tkchat.Chat, error) {
	uri := "/v1/chats"

	data := map[string]any{
		"customer_id": customerID.String(),
		"type":        string(talkType),
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/talks", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create talk: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &chat, nil
}

// TalkV1ChatDelete deletes a talk (soft delete)
func (r *requestHandler) TalkV1ChatDelete(ctx context.Context, talkID uuid.UUID) (*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", talkID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete talk: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talk")
	}

	return &chat, nil
}

// TalkV1ChatList gets a list of talks (simplified - for future expansion)
func (r *requestHandler) TalkV1ChatList(ctx context.Context, pageToken string, pageSize uint64) ([]*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats?page_token=%s&page_size=%d", pageToken, pageSize)

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/talks", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list talks: status %d", res.StatusCode)
	}

	var chats []*tkchat.Chat
	if err := json.Unmarshal(res.Data, &chats); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal talks")
	}

	return chats, nil
}
