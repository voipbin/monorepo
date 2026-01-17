package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	log "github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/talk"
)

func (h *listenHandler) processV1TalkChats(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	switch m.Method {
	case "POST":
		return h.v1TalkChatsPost(ctx, m)
	case "GET":
		return h.v1TalkChatsGet(ctx, m)
	default:
		return simpleResponse(405), nil
	}
}

func (h *listenHandler) v1TalkChatsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	var req struct {
		CustomerID string `json:"customer_id"`
		Type       string `json:"type"`
	}

	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Errorf("Failed to parse request: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	if customerID == uuid.Nil {
		return simpleResponse(400), nil
	}

	t, err := h.talkHandler.TalkCreate(ctx, customerID, talk.Type(req.Type))
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

func (h *listenHandler) v1TalkChatsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
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
		json.Unmarshal(m.Data, &filters)
	}

	// TODO: Convert filters to typed filters using utilhandler

	talks, err := h.talkHandler.TalkList(ctx, nil, pageToken, pageSize)
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

func (h *listenHandler) processV1TalkChatsID(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	matches := regV1TalkChatsID.FindStringSubmatch(m.URI)
	talkID := uuid.FromStringOrNil(matches[1])

	switch m.Method {
	case "GET":
		t, err := h.talkHandler.TalkGet(ctx, talkID)
		if err != nil {
			return simpleResponse(404), nil
		}
		data, _ := json.Marshal(t)
		return &commonsock.Response{
			StatusCode: 200,
			DataType:   "application/json",
			Data:       data,
		}, nil

	case "DELETE":
		t, err := h.talkHandler.TalkDelete(ctx, talkID)
		if err != nil {
			return simpleResponse(500), nil
		}
		data, _ := json.Marshal(t)
		return &commonsock.Response{
			StatusCode: 200,
			DataType:   "application/json",
			Data:       data,
		}, nil

	default:
		return simpleResponse(405), nil
	}
}
