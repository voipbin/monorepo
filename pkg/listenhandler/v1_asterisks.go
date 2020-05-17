package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// processV1AsterisksIDChannelsIDHealthPost handles /v1/asterisks/<id>/channels/<id>/health-check request
func (h *listenHandler) processV1AsterisksIDChannelsIDHealthPost(m *rabbitmq.Request) (*rabbitmq.Response, error) {
	ctx := context.Background()

	uriItems := strings.Split(m.URI, "/")
	if len(uriItems) < 7 {
		return nil, fmt.Errorf("wrong uri")
	}

	tmpAsteriskID := uriItems[3]
	channelID := uriItems[5]
	asteriskID, err := url.QueryUnescape(tmpAsteriskID)
	if err != nil {
		return nil, fmt.Errorf("could not unescape the asterisk id. err: %v", err)
	}

	type Data struct {
		RetryCount    int `json:"retry_count"`
		RetryCountMax int `json:"retry_count_max"`
		Delay         int `json:"delay"`
	}

	var data Data
	if err := json.Unmarshal([]byte(m.Data), &data); err != nil {
		return nil, err
	}

	channel, err := h.db.ChannelGet(ctx, asteriskID, channelID)
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"asterisk": asteriskID,
				"channel":  channelID,
			}).Errorf("Could not get the channel from the database. err: %v", err)
	}

	if channel.TMEnd != "" {
		logrus.WithFields(
			logrus.Fields{
				"asterisk": asteriskID,
				"channel":  channelID,
			}).Debug("The channel has hungup already. Stop to health-check.")
		return nil, nil
	}

	// send a channel heaclth check
	_, err = h.reqHandler.AstChannelGet(asteriskID, channelID)
	if err != nil {
		data.RetryCount++
	} else {
		data.RetryCount = 0
	}

	// todo: if the retry count is bigger than 2,
	// then generate fake-ChannelDestroyed event
	if data.RetryCount >= data.RetryCountMax {
		logrus.WithFields(
			logrus.Fields{
				"asterisk": asteriskID,
				"channel":  channelID,
			}).Info("Could not get channel info correctly. Terminating the channel.")
		return nil, nil
	}

	// send another health check.
	if err := h.reqHandler.CallChannelHealth(asteriskID, channelID, data.Delay, data.RetryCount, data.RetryCountMax); err != nil {
		return nil, err
	}

	return nil, nil
}
