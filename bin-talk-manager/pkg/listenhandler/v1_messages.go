package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/pkg/messagehandler"
)

func (h *listenHandler) processV1Messages(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	switch m.Method {
	case "POST":
		return h.v1MessagesPost(ctx, m)
	case "GET":
		return h.v1MessagesGet(ctx, m)
	default:
		return simpleResponse(405), nil
	}
}

func (h *listenHandler) v1MessagesPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	var req struct {
		CustomerID string  `json:"customer_id"`
		ChatID     string  `json:"chat_id"`
		ParentID   *string `json:"parent_id,omitempty"`
		OwnerType  string  `json:"owner_type"`
		OwnerID    string  `json:"owner_id"`
		Type       string  `json:"type"`
		Text       string  `json:"text"`
		Medias     string  `json:"medias"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	chatID := uuid.FromStringOrNil(req.ChatID)
	ownerID := uuid.FromStringOrNil(req.OwnerID)

	if customerID == uuid.Nil || chatID == uuid.Nil || ownerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		pid := uuid.FromStringOrNil(*req.ParentID)
		parentID = &pid
	}

	createReq := messagehandler.MessageCreateRequest{
		CustomerID: customerID,
		ChatID:     chatID,
		ParentID:   parentID,
		OwnerType:  req.OwnerType,
		OwnerID:    ownerID,
		Type:       req.Type,
		Text:       req.Text,
		Medias:     req.Medias,
	}

	msg, err := h.messageHandler.MessageCreate(ctx, createReq)
	if err != nil {
		log.Errorf("Failed to create message: %v", err)
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(msg)
	return &commonsock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1MessagesGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	u, _ := url.Parse(m.URI)

	// Parse pagination
	tmpSize, _ := strconv.Atoi(u.Query().Get("page_size"))
	pageSize := uint64(tmpSize)
	if pageSize == 0 {
		pageSize = 50
	}
	pageToken := u.Query().Get("page_token")

	// Parse filters from request body
	var filters map[string]any
	if m.Data != nil {
		json.Unmarshal(m.Data, &filters)
	}

	// TODO: Convert filters to typed filters using utilhandler

	messages, err := h.messageHandler.MessageList(ctx, nil, pageToken, pageSize)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(messages)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) processV1MessagesID(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1MessagesID.FindStringSubmatch(m.URI)
	messageID := uuid.FromStringOrNil(matches[1])

	switch m.Method {
	case "GET":
		msg, err := h.messageHandler.MessageGet(ctx, messageID)
		if err != nil {
			return simpleResponse(404), nil
		}
		data, _ := json.Marshal(msg)
		return &commonsock.Response{
			StatusCode: 200,
			DataType:   "application/json",
			Data:       data,
		}, nil

	case "DELETE":
		msg, err := h.messageHandler.MessageDelete(ctx, messageID)
		if err != nil {
			return simpleResponse(500), nil
		}
		data, _ := json.Marshal(msg)
		return &commonsock.Response{
			StatusCode: 200,
			DataType:   "application/json",
			Data:       data,
		}, nil

	default:
		return simpleResponse(405), nil
	}
}
