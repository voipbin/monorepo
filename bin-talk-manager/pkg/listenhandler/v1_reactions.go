package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
)

func (h *listenHandler) processV1MessagesIDReactions(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1MessagesIDReactions.FindStringSubmatch(m.URI)
	messageID := uuid.FromStringOrNil(matches[1])

	switch m.Method {
	case "POST":
		return h.v1MessagesIDReactionsPost(ctx, m, messageID)
	case "DELETE":
		return h.v1MessagesIDReactionsDelete(ctx, m, messageID)
	default:
		return simpleResponse(405), nil
	}
}

func (h *listenHandler) v1MessagesIDReactionsPost(ctx context.Context, m commonsock.Request, messageID uuid.UUID) (*commonsock.Response, error) {
	var req struct {
		OwnerType string `json:"owner_type"`
		OwnerID   string `json:"owner_id"`
		Reaction  string `json:"reaction"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	ownerID := uuid.FromStringOrNil(req.OwnerID)
	if ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	msg, err := h.reactionHandler.ReactionAdd(ctx, messageID, req.Reaction, req.OwnerType, ownerID)
	if err != nil {
		log.Errorf("Failed to add reaction: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(msg)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1MessagesIDReactionsDelete(ctx context.Context, m commonsock.Request, messageID uuid.UUID) (*commonsock.Response, error) {
	var req struct {
		OwnerType string `json:"owner_type"`
		OwnerID   string `json:"owner_id"`
		Reaction  string `json:"reaction"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	ownerID := uuid.FromStringOrNil(req.OwnerID)
	if ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	msg, err := h.reactionHandler.ReactionRemove(ctx, messageID, req.Reaction, req.OwnerType, ownerID)
	if err != nil {
		log.Errorf("Failed to remove reaction: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(msg)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
