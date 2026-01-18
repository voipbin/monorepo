package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"monorepo/bin-common-handler/models/sock"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// TalkV1ChatGet gets a chat by ID
func (r *requestHandler) TalkV1ChatGet(ctx context.Context, chatID uuid.UUID) (*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", chatID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/chats", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get chat: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal chat")
	}

	return &chat, nil
}

// TalkV1ChatCreate creates a new chat
func (r *requestHandler) TalkV1ChatCreate(ctx context.Context, customerID uuid.UUID, chatType tkchat.Type, name string, detail string, creatorType string, creatorID uuid.UUID, participants []tkparticipant.ParticipantInput) (*tkchat.Chat, error) {
	uri := "/v1/chats"

	data := map[string]any{
		"customer_id":  customerID.String(),
		"type":         string(chatType),
		"name":         name,
		"detail":       detail,
		"creator_type": creatorType,
		"creator_id":   creatorID.String(),
	}

	// Add participants if provided
	if len(participants) > 0 {
		participantList := make([]map[string]string, 0, len(participants))
		for _, p := range participants {
			participantList = append(participantList, map[string]string{
				"owner_type": p.OwnerType,
				"owner_id":   p.OwnerID.String(),
			})
		}
		data["participants"] = participantList
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal request")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodPost, "talk/chats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 && res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to create chat: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal chat")
	}

	return &chat, nil
}

// TalkV1ChatDelete deletes a chat (soft delete)
func (r *requestHandler) TalkV1ChatDelete(ctx context.Context, chatID uuid.UUID) (*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", chatID.String())

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodDelete, "talk/chats", requestTimeoutDefault, 0, "", nil)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to delete chat: status %d", res.StatusCode)
	}

	var chat tkchat.Chat
	if err := json.Unmarshal(res.Data, &chat); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal chat")
	}

	return &chat, nil
}

// TalkV1ChatList gets a list of chats with optional filters
func (r *requestHandler) TalkV1ChatList(ctx context.Context, filters map[string]any, pageToken string, pageSize uint64) ([]*tkchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats?page_token=%s&page_size=%d", pageToken, pageSize)

	var data []byte
	var err error

	// Marshal filters to JSON for request body if provided
	if len(filters) > 0 {
		data, err = json.Marshal(filters)
		if err != nil {
			return nil, errors.Wrap(err, "could not marshal filters")
		}
	} else {
		data = []byte("{}")
	}

	res, err := r.sendRequestTalk(ctx, uri, sock.RequestMethodGet, "talk/chats", requestTimeoutDefault, 0, ContentTypeJSON, data)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to list chats: status %d", res.StatusCode)
	}

	var chats []*tkchat.Chat
	if err := json.Unmarshal(res.Data, &chats); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal chats")
	}

	return chats, nil
}
