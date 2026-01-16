package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/pkg/listenhandler/models/request"
)

// v1ChatroomsGet handles /v1/chatrooms GET request
func (h *listenHandler) v1ChatroomsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1ChatroomsGet",
		"request": m,
	})

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// parse the pagination params from URI
	tmpSize, _ := strconv.Atoi(u.Query().Get(PageSize))
	pageSize := uint64(tmpSize)
	pageToken := u.Query().Get(PageToken)

	// Parse filters from request data (body)
	var filters map[string]any
	if len(m.Data) > 0 {
		if err := json.Unmarshal(m.Data, &filters); err != nil {
			log.Errorf("Could not unmarshal filters. err: %v", err)
			return nil, fmt.Errorf("could not unmarshal filters: %w", err)
		}
	}

	log.WithFields(logrus.Fields{
		"filters":          filters,
		"filters_raw_data": string(m.Data),
	}).Debug("v1ChatroomsGet: Parsed filters from request body")

	// Convert string map to typed field map
	typedFilters, err := chatroom.ConvertStringMapToFieldMap(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, fmt.Errorf("could not convert filters: %w", err)
	}

	log.WithFields(logrus.Fields{
		"typed_filters": typedFilters,
	}).Debug("v1ChatroomsGet: Converted filters to typed field map (check UUID types)")

	tmp, err := h.chatroomHandler.List(ctx, pageToken, pageSize, typedFilters)
	if err != nil {
		log.Errorf("Could not get chatrooms. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ChatroomsIDGet handles /v1/chatrooms/{id} GET request
func (h *listenHandler) v1ChatroomsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsIDGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	chatroomID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.chatroomHandler.Get(ctx, chatroomID)
	if err != nil {
		log.Errorf("Could not get chatroom info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ChatroomsIDPut handles /v1/chatrooms/{id} PUT request
func (h *listenHandler) v1ChatroomsIDPut(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsIDPut",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/chatrooms/2d8d416a-bc5b-11ee-bcaa-8728b23e22f1"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])

	var req request.V1DataChatroomsIDPut
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	tmp, err := h.chatroomHandler.UpdateBasicInfo(ctx, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the chatroom info. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}

// v1ChatroomsIDDelete handles /v1/chatrooms/{id} Delete request
func (h *listenHandler) v1ChatroomsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1ChatroomsIDDelete",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	// "/v1/chatrooms/a6f4eae8-8a74-11ea-af75-3f1e61b9a236"
	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("chatroom_id", id)

	tmp, err := h.chatroomHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the chatroom. err: %v", err)
		return nil, err
	}

	data, err := json.Marshal(tmp)
	if err != nil {
		log.Errorf("Could not marshal the res. err: %v", err)
		return nil, err
	}

	res := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}

	return res, nil
}
