package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) v1ChatsIDParticipantsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsIDParticipants.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	var req struct {
		CustomerID string `json:"customer_id"`
		OwnerType  string `json:"owner_type"`
		OwnerID    string `json:"owner_id"`
	}

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

	var req struct {
		CustomerID string `json:"customer_id"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)

	if customerID == uuid.Nil || chatID == uuid.Nil {
		return simpleResponse(400), nil
	}

	participants, err := h.participantHandler.ParticipantList(ctx, customerID, chatID)
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
	participantID := uuid.FromStringOrNil(matches[2])

	var req struct {
		CustomerID string `json:"customer_id"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)

	if customerID == uuid.Nil || participantID == uuid.Nil {
		return simpleResponse(400), nil
	}

	err = h.participantHandler.ParticipantRemove(ctx, customerID, participantID)
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
