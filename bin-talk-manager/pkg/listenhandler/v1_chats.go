package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/chat"
)

func (h *listenHandler) v1ChatsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	var req struct {
		CustomerID string `json:"customer_id"`
		Type       string `json:"type"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		logrus.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	if customerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	t, err := h.chatHandler.ChatCreate(ctx, customerID, chat.Type(req.Type))
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(t)
	return &commonsock.Response{
		StatusCode: 201,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1ChatsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
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
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &filters); err != nil {
			logrus.Errorf("Failed to parse filters: %v", err)
			return simpleResponse(400), nil
		}
	}

	// TODO: Convert filters to typed filters using utilhandler

	talks, err := h.chatHandler.ChatList(ctx, nil, pageToken, pageSize)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(talks)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1ChatsIDGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsID.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	t, err := h.chatHandler.ChatGet(ctx, chatID)
	if err != nil {
		return simpleResponse(404), nil
	}

	data, _ := json.Marshal(t)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1ChatsIDDelete(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1ChatsID.FindStringSubmatch(m.URI)
	chatID := uuid.FromStringOrNil(matches[1])

	t, err := h.chatHandler.ChatDelete(ctx, chatID)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(t)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
