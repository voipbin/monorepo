package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
)

// processV1ChannelsIDHealthPost handles /v1/channels/<id>/health-check request
func (h *listenHandler) processV1ChannelsIDHealthPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 5 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	channelID := uriItems[3]

	var data request.V1DataChannelsIDHealth
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log := logrus.WithFields(
		logrus.Fields{
			"channel_id": channelID,
		})
	log.Debugf("Received channel health-check request. channel_id: %s, retry: %d, retry_max: %d, delay: %d", channelID, data.RetryCount, data.RetryCountMax, data.Delay)

	h.channelHandler.HealthCheck(ctx, channelID, data.RetryCount, data.RetryCountMax, data.Delay)

	return nil, nil
}
