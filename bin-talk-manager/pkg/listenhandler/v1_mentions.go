package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	commonsock "monorepo/bin-common-handler/models/sock"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/listenhandler/models/request"
)

// v1MentionsGet handles GET /v1/mentions. It returns the messages that mention the agent.
func (h *listenHandler) v1MentionsGet(ctx context.Context, m commonsock.Request) (*commonsock.Response, error) {
	log := logrus.WithFields(logrus.Fields{"func": "v1MentionsGet", "request": m})

	u, _ := url.Parse(m.URI)

	tmpSize, _ := strconv.Atoi(u.Query().Get("page_size"))
	pageSize := uint64(tmpSize)
	if pageSize == 0 {
		pageSize = 50
	}
	pageToken := u.Query().Get("page_token")

	var req request.V1DataMentionsGet
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		return simpleResponse(400), nil
	}

	customerID := uuid.FromStringOrNil(req.CustomerID)
	agentID := uuid.FromStringOrNil(req.AgentID)
	if customerID == uuid.Nil || agentID == uuid.Nil {
		return simpleResponse(400), nil
	}

	messages, err := h.messageHandler.MentionList(ctx, customerID, agentID, pageToken, pageSize)
	if err != nil {
		log.Errorf("Could not list the mentions. err: %v", err)
		return errorResponse(err), nil
	}
	if messages == nil {
		messages = []*message.Message{}
	}

	data, _ := json.Marshal(messages)
	return &commonsock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}
