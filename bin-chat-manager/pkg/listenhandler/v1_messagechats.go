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

	"monorepo/bin-chat-manager/models/media"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/pkg/listenhandler/models/request"
)

// v1MessagechatsPost handles /v1/messagechats POST request
// creates a new messagechat with given data and return the created messagechat info.
func (h *listenHandler) v1MessagechatsPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1MessagechatsPost",
	})
	log.WithField("request", m).Debug("Received request.")

	var req request.V1DataMessagechatsPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not marshal the data. err: %v", err)
		return nil, err
	}

	if req.Medias == nil {
		req.Medias = []media.Media{}
	}

	// create
	tmp, err := h.messagechatHandler.Create(
		ctx,
		req.CustomerID,
		req.ChatID,
		&req.Source,
		req.MessageType,
		req.Text,
		req.Medias,
	)
	if err != nil {
		log.Errorf("Could not create a new messagechat. err: %v", err)
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

// v1MessagechatsGet handles /v1/messagechats GET request
func (h *listenHandler) v1MessagechatsGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "v1MessagechatsGet",
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
	}).Debug("v1MessagechatsGet: Parsed filters from request body")

	// Convert string map to typed field map
	typedFilters, err := messagechat.ConvertStringMapToFieldMap(filters)
	if err != nil {
		log.Errorf("Could not convert filters. err: %v", err)
		return nil, fmt.Errorf("could not convert filters: %w", err)
	}

	log.WithFields(logrus.Fields{
		"typed_filters": typedFilters,
	}).Debug("v1MessagechatsGet: Converted filters to typed field map (check UUID types)")

	tmp, err := h.messagechatHandler.List(ctx, pageToken, pageSize, typedFilters)
	if err != nil {
		log.Errorf("Could not get messagechats. err: %v", err)
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

// v1MessagechatsIDGet handles /v1/messagechats/{id} GET request
func (h *listenHandler) v1MessagechatsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1MessagechatsIDGet",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	messagechatID := uuid.FromStringOrNil(tmpVals[3])

	tmp, err := h.messagechatHandler.Get(ctx, messagechatID)
	if err != nil {
		log.Errorf("Could not get messagechat info. err: %v", err)
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

// v1MessagechatsIDDelete handles /v1/messagechats/{id} Delete request
func (h *listenHandler) v1MessagechatsIDDelete(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1MessagechatsIDDelete",
	})
	log.WithField("request", m).Debug("Received request.")

	u, err := url.Parse(m.URI)
	if err != nil {
		return nil, err
	}

	tmpVals := strings.Split(u.Path, "/")
	id := uuid.FromStringOrNil(tmpVals[3])
	log = log.WithField("chat_id", id)

	tmp, err := h.messagechatHandler.Delete(ctx, id)
	if err != nil {
		log.Errorf("Could not delete the messagechat. err: %v", err)
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
