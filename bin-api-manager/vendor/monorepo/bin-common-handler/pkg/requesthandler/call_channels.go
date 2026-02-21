package requesthandler

import (
	"context"
	"fmt"

	cmchannel "monorepo/bin-call-manager/models/channel"
	"monorepo/bin-common-handler/models/sock"
)

// CallV1ChannelGet sends a request to the call-manager to get a channel by ID.
func (r *requestHandler) CallV1ChannelGet(ctx context.Context, channelID string) (*cmchannel.Channel, error) {
	uri := fmt.Sprintf("/v1/channels/%s", channelID)

	tmp, err := r.sendRequestCall(ctx, uri, sock.RequestMethodGet, "call/channels", requestTimeoutDefault, 0, ContentTypeJSON, nil)
	if err != nil {
		return nil, err
	}

	var res cmchannel.Channel
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}
