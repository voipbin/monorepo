package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) processV1TalksIDParticipants(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
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

func (h *listenHandler) v1TalksIDParticipantsPost(ctx context.Context, m commonsock.Request, chatID uuid.UUID) (*commonsock.Response, error) {
	var req struct {
		CustomerID string `json:"customer_id"`
		OwnerType  string `json:"owner_type"`
		OwnerID    string `json:"owner_id"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	ownerID := uuid.FromStringOrNil(req.OwnerID)

	if customerID == uuid.Nil || ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	participant, err := h.participantHandler.ParticipantAdd(ctx, customerID, chatID, ownerID, req.OwnerType)
	if err != nil {
		log.Errorf("Failed to create participant: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participant)
	return &commonsock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1TalksIDParticipantsGet(ctx context.Context, m commonsock.Request, chatID uuid.UUID) (*commonsock.Response, error) {
	// Parse customer_id from request body
	var req struct {
		CustomerID string `json:"customer_id"`
	}
	json.Unmarshal(m.Data, &req)
	customerID := uuid.FromStringOrNil(req.CustomerID)

	participants, err := h.participantHandler.ParticipantList(ctx, customerID, chatID)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(participants)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) processV1TalksIDParticipantsID(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1TalksIDParticipantsID.FindStringSubmatch(m.URI)
	participantID := uuid.FromStringOrNil(matches[2])

	switch m.Method {
	case "DELETE":
		// Parse customer_id from request body
		var req struct {
			CustomerID string `json:"customer_id"`
		}
		json.Unmarshal(m.Data, &req)
		customerID := uuid.FromStringOrNil(req.CustomerID)

		err := h.participantHandler.ParticipantRemove(ctx, customerID, participantID)
		if err != nil {
			return simpleResponse(500), nil
		}
		return &commonsock.Response{
			StatusCode: 204,
			DataType:   "application/json",
			Data:       json.RawMessage("{}"),
		}, nil

	default:
		return simpleResponse(405), nil
	}
}
