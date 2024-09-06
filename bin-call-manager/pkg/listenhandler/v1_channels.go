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
