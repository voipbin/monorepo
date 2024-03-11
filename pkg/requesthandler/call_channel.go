package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmrequest "gitlab.com/voipbin/bin-manager/call-manager.git/pkg/listenhandler/models/request"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// CallV1ChannelHealth sends the request to the call-manager for channel health-check
func (r *requestHandler) CallV1ChannelHealth(ctx context.Context, channelID string, delay, retryCount int) error {
	uri := fmt.Sprintf("/v1/channels/%s/health-check", channelID)

	m, err := json.Marshal(cmrequest.V1DataChannelsIDHealth{
		RetryCount: retryCount,
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
