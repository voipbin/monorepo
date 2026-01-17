package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/pkg/participanthandler"
)

func (h *listenHandler) processV1TalksIDParticipants(ctx context.Context, m commonsock.Request) (commonsock.Response, error) {
	matches := regV1TalksIDParticipants.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	switch m.Method {
	case "POST":
		return h.v1TalksIDParticipantsPost(ctx, m, chatID)
	case "GET":
		return h.v1TalksIDParticipantsGet(ctx, m, chatID)
	default:
		return simpleResponse(405), nil
	}
}

func (h *listenHandler) v1TalksIDParticipantsPost(ctx context.Context, m commonsock.Request, chatID uuid.UUID) (commonsock.Response, error) {
	var req struct {
		CustomerID string `json:"customer_id"`
		OwnerType  string `json:"owner_type"`
		OwnerID    string `json:"owner_id"`
	}

	err := json.Unmarshal(m.Data.([]byte), &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	ownerID := uuid.FromStringOrNil(req.OwnerID)

	if customerID == uuid.Nil || ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	createReq := participanthandler.ParticipantCreateRequest{
		CustomerID: customerID,
		ChatID:     chatID,
		OwnerType:  req.OwnerType,
		OwnerID:    ownerID,
	}

	participant, err := h.participantHandler.ParticipantCreate(ctx, createReq)
	if err != nil {
		log.Errorf("Failed to create participant: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participant)
	return commonsock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       string(data),
	}, nil
}

func (h *listenHandler) v1TalksIDParticipantsGet(ctx context.Context, m commonsock.Request, chatID uuid.UUID) (commonsock.Response, error) {
	participants, err := h.participantHandler.ParticipantListByChatID(ctx, chatID)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participants)
	return commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       string(data),
	}, nil
}

func (h *listenHandler) processV1TalksIDParticipantsID(ctx context.Context, m commonsock.Request) (commonsock.Response, error) {
	matches := regV1TalksIDParticipantsID.FindStringSubmatch(m.URI)
	participantID := uuid.FromStringOrNil(matches[2])

	switch m.Method {
	case "DELETE":
		participant, err := h.participantHandler.ParticipantDelete(ctx, participantID)
		if err != nil {
			return simpleResponse(500), nil
		}
		data, _ := json.Marshal(participant)
		return commonsock.Response{
			StatusCode: 200,
			DataType:   "application/json",
			Data:       string(data),
		}, nil

	default:
		return simpleResponse(405), nil
	}
}
