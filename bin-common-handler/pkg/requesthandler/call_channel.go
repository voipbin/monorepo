package requesthandler

import (
	"context"
	"encoding/json"
	"fmt"

	cmrequest "monorepo/bin-call-manager/pkg/listenhandler/models/request"
	"monorepo/bin-common-handler/models/sock"
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

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodPost, "call/channels/health", requestTimeoutDefault, delay, ContentTypeJSON, m)
	if err != nil {
		return err
	}

	if errParse := parseResponse(tmp, nil); errParse != nil {
		return errParse
	}

	return nil
}
