package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	chatchat "monorepo/bin-chat-manager/models/chat"
	chatrequest "monorepo/bin-chat-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// ChatV1ChatCreate sends a request to chat-manager
// to creating a chat.
// it returns created chat if it succeed.
func (r *requestHandler) ChatV1ChatCreate(
	ctx context.Context,
	customerID uuid.UUID,
	chatType chatchat.Type,
	roomOwnerID uuid.UUID,
	participantIDs []uuid.UUID,
	name string,
	detail string,
) (*chatchat.Chat, error) {
	uri := "/v1/chats"

	data := &chatrequest.V1DataChatsPost{
		CustomerID:     customerID,
		Type:           chatType,
		RoomOwnerID:    roomOwnerID,
		ParticipantIDs: participantIDs,
		Name:           name,
		Detail:         detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodPost, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatGet sends a request to chat-manager
// to getting a chat.
// it returns given chat id's chat if it succeed.
func (r *requestHandler) ChatV1ChatGet(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", chatID)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodGet, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatGets sends a request to chat-manager
// to getting a list of chat info.
// it returns detail list of chat info if it succeed.
func (r *requestHandler) ChatV1ChatList(ctx context.Context, pageToken string, pageSize uint64, filters map[chatchat.Field]any) ([]chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats?page_token=%s&page_size=%d", url.QueryEscape(pageToken), pageSize)

	m, err := json.Marshal(filters)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal filters")
	}

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodGet, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res []chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return res, nil
}

// ChatV1ChatDelete sends a request to chat-manager
// to delete the chat.
// it returns error if it went wrong.
func (r *requestHandler) ChatV1ChatDelete(ctx context.Context, chatID uuid.UUID) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", chatID)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodDelete, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatUpdateBasicInfo sends a request to chat-manager
// to update the chat's basic info.
func (r *requestHandler) ChatV1ChatUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s", id)

	data := &chatrequest.V1DataChatsIDPut{
		Name:   name,
		Detail: detail,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodPut, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatUpdateRoomOwnerID sends a request to chat-manager
// to update the chat's owner id.
func (r *requestHandler) ChatV1ChatUpdateRoomOwnerID(ctx context.Context, id uuid.UUID, roomOwnerID uuid.UUID) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s/room_owner_id", id)

	data := &chatrequest.V1DataChatsIDRoomOwnerIDPut{
		RoomOwnerID: roomOwnerID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodPut, "chat/chats/<chat-id>/room_owner_id", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatAddParticipantID sends a request to chat-manager
// to add the participant id to the chat.
func (r *requestHandler) ChatV1ChatAddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s/participant_ids", id)

	data := &chatrequest.V1DataChatsIDParticipantIDsPost{
		ParticipantID: participantID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodPost, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

// ChatV1ChatRemoveParticipantID sends a request to chat-manager
// to remove the participant id from the chat.
func (r *requestHandler) ChatV1ChatRemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatchat.Chat, error) {
	uri := fmt.Sprintf("/v1/chats/%s/participant_ids/%s", id, participantID)

	tmp, err := r.sendRequestChat(ctx, uri, sock.RequestMethodDelete, "chat/chats", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res chatchat.Chat
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
