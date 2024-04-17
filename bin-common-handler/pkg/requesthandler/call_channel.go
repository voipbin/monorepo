package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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
