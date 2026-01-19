package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

func (h *listenHandler) v1ChatsIDParticipantsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsIDParticipants.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	var req request.V1DataChatsIDParticipantsPost

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	ownerID := uuid.FromStringOrNil(req.OwnerID)

	if customerID == uuid.Nil || chatID == uuid.Nil || ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	participant, err := h.participantHandler.ParticipantAdd(ctx, customerID, chatID, ownerID, req.OwnerType)
	if err != nil {
		logrus.Errorf("Failed to add participant: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participant)
	return &commonsock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1ChatsIDParticipantsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsIDParticipants.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	if chatID == uuid.Nil {
		return simpleResponse(400), nil
	}

	// Get chat to retrieve customer_id for authorization
	chat, err := h.chatHandler.ChatGet(ctx, chatID)
	if err != nil {
		logrus.Errorf("Failed to get chat: %v", err)
		return simpleResponse(404), nil
	}

	participants, err := h.participantHandler.ParticipantList(ctx, chat.CustomerID, chatID)
	if err != nil {
		logrus.Errorf("Failed to list participants: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participants)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1ChatsIDParticipantsIDDelete(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsIDParticipantsID.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])
	participantID := uuid.FromStringOrNil(matches[2])

	if chatID == uuid.Nil || participantID == uuid.Nil {
		return simpleResponse(400), nil
	}

	// Get chat to retrieve customer_id for authorization
	chat, err := h.chatHandler.ChatGet(ctx, chatID)
	if err != nil {
		logrus.Errorf("Failed to get chat: %v", err)
		return simpleResponse(404), nil
	}

	err = h.participantHandler.ParticipantRemove(ctx, chat.CustomerID, participantID)
	if err != nil {
		logrus.Errorf("Failed to remove participant: %v", err)
		return simpleResponse(500), nil
	}

	return &commonsock.Response{
		StatusCode: 204,
		DataType:   "application/json",
		Data:       json.RawMessage("{}"),
	}, nil
}
