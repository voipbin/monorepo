package listenhandler

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

func (h *listenHandler) v1MessagesIDReactionsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1MessagesIDReactions.FindStringSubmatch(m.URI)
	messageID := uuid.FromStringOrNil(matches[1])

	var req request.V1DataMessagesIDReactionsPost

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	ownerID := uuid.FromStringOrNil(req.OwnerID)
	if ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	msg, err := h.reactionHandler.ReactionAdd(ctx, messageID, req.Reaction, req.OwnerType, ownerID)
	if err != nil {
		logrus.Errorf("Failed to add reaction: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(msg)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1MessagesIDReactionsDelete(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1MessagesIDReactions.FindStringSubmatch(m.URI)
	messageID := uuid.FromStringOrNil(matches[1])

	var req request.V1DataMessagesIDReactionsPost

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	ownerID := uuid.FromStringOrNil(req.OwnerID)
	if ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	msg, err := h.reactionHandler.ReactionRemove(ctx, messageID, req.Reaction, req.OwnerType, ownerID)
	if err != nil {
		logrus.Errorf("Failed to remove reaction: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(msg)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
