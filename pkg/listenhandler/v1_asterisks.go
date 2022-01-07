package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
)

// processV1AsterisksIDChannelsIDHealthPost handles /v1/asterisks/<id>/channels/<id>/health-check request
func (h *listenHandler) processV1AsterisksIDChannelsIDHealthPost(ctx context.Context, m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {
	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 7 {
		return simpleResponse(400), fmt.Errorf("wrong uri")
	}

	tmpAsteriskID := uriItems[3]
	channelID := uriItems[5]
	asteriskID, err := url.QueryUnescape(tmpAsteriskID)
	if err != nil {
		return nil, fmt.Errorf("could not unescape the asterisk id. err: %v", err)
	}

	var data request.V1DataAsterisksIDChannelsIDHealth
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}
	log := logrus.WithFields(
		logrus.Fields{
			"asterisk": asteriskID,
			"channel":  channelID,
		})
	log.Debugf("Received health-check request. retry: %d, retry_max: %d, delay: %d", data.RetryCount, data.RetryCountMax, data.Delay)

	h.callHandler.ChannelHealthCheck(ctx, asteriskID, channelID, data.RetryCount, data.RetryCountMax, data.Delay)

	return nil, nil
}
