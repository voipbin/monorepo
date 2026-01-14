package listenhandler

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

// processV1MessagesGet handles GET /v1/messages request
func (h *listenHandler) processV1MessagesGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1MessagesGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		log.Errorf("Could not parse the request uri. err: %v", err)
		return simpleResponse(400), nil
	}

	// parse the pagination params
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// get aicall id (still from URL for backward compatibility)
	aicallID := uuid.FromStringOrNil(u.Query().Get("aicall_id"))

	// get filters from request body
	tmpFilters, err := utilhandler.ParseFiltersFromRequestBody(m.Data)
	if err != nil {
		log.Errorf("Could not parse filters. err: %v", err)
		return simpleResponse(400), nil
	}

	// convert to typed filters
	typedFilters, err := utilhandler.ConvertFilters[message.FieldStruct, message.Field](message.FieldStruct{}, tmpFilters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return simpleResponse(400), nil
	}

	log = log.WithFields(logrus.Fields{
		"aicall_id": aicallID,
		"size":      pageSize,
		"token":     pageToken,
		"filters":   typedFilters,
	})

	tmp, err := h.messageHandler.Gets(ctx, aicallID, pageSize, pageToken, typedFilters)
	if err != nil {
		log.Debugf("Could not get messages. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Debugf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesPost handles POST /v1/messages request
func (h *listenHandler) processV1MessagesPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1MessagesPost",
		"request": m,
	})

	var req request.V1DataMessagesPost
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not unmarshal the requested data. err: %v", err)
		return simpleResponse(400), nil
	}

	tmp, err := h.aicallHandler.Send(ctx, req.AIcallID, req.Role, req.Content, req.RunImmediately, req.AudioResponse)
	if err != nil {
		log.Errorf("Could not create ai. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// processV1MessagesIDGet handles POST /v1/messages/<message-id> request
func (h *listenHandler) processV1MessagesIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"handler": "processV1MessagesIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		log.Errorf("Wrong uri item count. uri_items: %d", len(uriItems))
		return simpleResponse(400), nil
	}
	id := uuid.FromStringOrNil(uriItems[3])

	tmp, err := h.messageHandler.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not create ai. err: %v", err)
		return simpleResponse(500), nil
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the response message. message: %v, err: %v", tmp, err)
		return simpleResponse(500), nil
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
