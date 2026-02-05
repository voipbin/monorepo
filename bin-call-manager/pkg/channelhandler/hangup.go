package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// Hangup deletes the channel.
func (h *channelHandler) Hangup(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	res, err := h.Delete(ctx, id, cause)
	if err != nil {
		return nil, errors.Wrapf(err, "could not delete the channel with id: %s", id)
	}

	return res, nil
}

// HangingUp starts the hangup process
func (h *channelHandler) HangingUp(ctx context.Context, id string, cause ari.ChannelCause) (*channel.Channel, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get channel info with id: %s", id)
	}

	if res.TMDelete != nil {
		// already hungup nothing to do
		return res, nil
	}

	if errHangup := h.HangingUpWithAsteriskID(ctx, res.AsteriskID, res.ID, cause); errHangup != nil {
		return nil, errors.Wrapf(errHangup, "could not hangup the channel with asteriskID: %s, channelID: %s", res.AsteriskID, res.ID)
	}

	return res, nil
}

// HangingUpWithAsteriskID starts the hangup process
func (h *channelHandler) HangingUpWithAsteriskID(ctx context.Context, asteriskID string, id string, cause ari.ChannelCause) error {
	if errHangup := h.reqHandler.AstChannelHangup(ctx, asteriskID, id, cause, 0); errHangup != nil {
		if errors.Cause(errHangup) == requesthandler.ErrNotFound {
			// channel doesn't exist. consider it hungup already.
			return nil
		}

		return errors.Wrapf(errHangup, "could not hangup the channel with asteriskID: %s, channelID: %s", asteriskID, id)
	}

	return nil
}

// HangingUpWithDelay starts the hangup process
func (h *channelHandler) HangingUpWithDelay(ctx context.Context, id string, cause ari.ChannelCause, delay int) (*channel.Channel, error) {
	res, err := h.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get channel info with id: %s", id)
	}

	if res.TMDelete != nil {
		// already hungup nothing to do
		return nil, fmt.Errorf("already hungup")
	}

	if errHangup := h.reqHandler.AstChannelHangup(ctx, res.AsteriskID, id, cause, delay); errHangup != nil {
		return nil, errors.Wrapf(errHangup, "could not hangup the channel with asteriskID: %s, channelID: %s", res.AsteriskID, res.ID)
	}

	return res, nil
}
