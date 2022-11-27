package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CallV1ChannelHealth sends the request for channel health-check
func (r *requestHandler) CallV1ChannelHealth(ctx context.Context, channelID string, delay, retryCount, retryCountMax int) error {
	uri := fmt.Sprintf("/v1/channels/%s/health-check", channelID)

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

	res, err := r.sendRequestCall(ctx, uri, rabbitmqhandler.RequestMethodPost, resourceCallChannelsHealth, requestTimeoutDefault, delay, ContentTypeJSON, m)
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
