package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/chat"
	"monorepo/bin-talk-manager/models/participant"
)

func (h *listenHandler) v1ChatsPost(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	var req struct {
		CustomerID   string `json:"customer_id"`
		Type         string `json:"type"`
		Name         string `json:"name"`
		Detail       string `json:"detail"`
		CreatorType  string `json:"creator_type"`
		CreatorID    string `json:"creator_id"`
		Participants []struct {
			OwnerType string `json:"owner_type"`
			OwnerID   string `json:"owner_id"`
		} `json:"participants"`
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

	creatorID := uuid.FromStringOrNil(req.CreatorID)

	// Parse participants
	var participants []participant.ParticipantInput
	for _, p := range req.Participants {
		ownerID := uuid.FromStringOrNil(p.OwnerID)
		if ownerID == uuid.Nil {
			logrus.Errorf("Invalid participant owner_id: %s", p.OwnerID)
			return simpleResponse(400), nil
		}
		participants = append(participants, participant.ParticipantInput{
			OwnerType: p.OwnerType,
			OwnerID:   ownerID,
		})
	}

	t, err := h.chatHandler.ChatCreate(ctx, customerID, chat.Type(req.Type), req.Name, req.Detail, req.CreatorType, creatorID, participants)
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

	// Parse filters from request body using utilhandler pattern
	tmpFilters, err := h.utilHandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		logrus.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// Convert to typed filters
	typedFilters, err := commonutil.ConvertFilters[chat.FieldStruct, chat.Field](
		chat.FieldStruct{},
		tmpFilters,
	)
	if err != nil {
		logrus.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	chats, err := h.chatHandler.ChatList(ctx, typedFilters, pageToken, pageSize)
	if err != nil {
		return simpleResponse(500), nil
	}

	data, _ := json.Marshal(chats)
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
