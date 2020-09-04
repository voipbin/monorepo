package requesthandler

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

// CallChannelHealth sends the request for channel health-check
func (r *requestHandler) CallChannelHealth(asteriskID, channelID string, delay, retryCount, retryCountMax int) error {
	encodeAsteriskID := url.QueryEscape(asteriskID)
	uri := fmt.Sprintf("/v1/asterisks/%s/channels/%s/health-check", encodeAsteriskID, channelID)

	type Data struct {
		RetryCount    int `json:"retry_count"`
		RetryCountMax int `json:"retry_count_max"`
		Delay         int `json:"delay"`
	}

	m, err := json.Marshal(Data{
		retryCount,
		retryCountMax,
		delay,
	})
	if err != nil {
		return err
	}

	res, err := r.sendRequestCall(uri, rabbitmq.RequestMethodPost, resourceCallChannelsHealth, requestTimeoutDefault, delay, ContentTypeJSON, m)
	switch {
	case err != nil:
		return err
	case res == nil:
		return nil
	case res.StatusCode > 299:
		return fmt.Errorf("response code: %d", res.StatusCode)
	}

	return nil
}
