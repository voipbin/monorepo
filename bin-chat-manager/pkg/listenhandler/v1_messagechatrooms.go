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

	"monorepo/bin-chat-manager/models/messagechatroom"
)

// v1MessagechatroomsGet handles /v1/messagechatrooms GET request
func (h *listenHandler) v1MessagechatroomsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1MessagechatroomsGet",
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
	}).Debug("v1MessagechatroomsGet: Parsed filters from request body")

	// Convert string map to typed field map
	typedFilters, err := messagechatroom.ConvertStringMapToFieldMap(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, fmt.Errorf("could not convert filters: %w", err)
	}

	log.WithFields(logrus.Fields{
		"typed_filters": typedFilters,
	}).Debug("v1MessagechatroomsGet: Converted filters to typed field map (check UUID types)")

	tmp, err := h.messagechatroomHandler.List(ctx, pageToken, pageSize, typedFilters)
	if err != nil {
		log.Errorf("Could not get messagechatrooms. err: %v", err)
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

// v1MessagechatroomsIDGet handles /v1/messagechatrooms/{id} GET request
func (h *listenHandler) v1MessagechatroomsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1MessagechatroomsIDGet",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	messagechatroomID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.messagechatroomHandler.Get(ctx, messagechatroomID)
	if err != nil {
		log.Errorf("Could not get messagechatroom info. err: %v", err)
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

// v1MessagechatroomsIDDelete handles /v1/messagechatrooms/{id} Delete request
func (h *listenHandler) v1MessagechatroomsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "v1MessagechatroomsIDDelete",
		},
	)
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("messagechatroom_id", id)

	tmp, err := h.messagechatroomHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the messagechatroom. err: %v", err)
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
