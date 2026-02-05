package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/listenhandler/models/request"
)

// processV1ChannelsIDGet handles GET /v1/channels/<id> request
func (h *listenHandler) processV1ChannelsIDGet(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ChannelsIDGet",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 4 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	channelID := uriItems[3]

	res, err := h.channelHandler.Get(ctx, channelID)
	if err != nil {
		log.Errorf("Could not get channel. err: %v", err)
		return simpleResponse(404), nil
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Errorf("Could not marshal response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

// processV1ChannelsIDHealthPost handles /v1/channels/<id>/health-check request
func (h *listenHandler) processV1ChannelsIDHealthPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processV1ChannelsIDHealthPost",
		"request": m,
	})

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	channelID := uriItems[3]

	var req request.V1DataChannelsIDHealth
	if err := json.Unmarshal([]byte(m.Data), &req); err != nil {
		log.Errorf("Could not marshal the request. err: %v", err)
		return nil, err
	}

	h.channelHandler.HealthCheck(ctx, channelID, req.RetryCount)

	return nil, nil
}
