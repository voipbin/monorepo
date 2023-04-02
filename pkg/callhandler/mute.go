package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
)

// MuteOn moh the call
func (h *callHandler) MuteOn(ctx context.Context, id uuid.UUID, direction call.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "MuteOn",
		"call_id":   id,
		"direction": direction,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if direction == call.MuteDirectionNone {
		direction = call.MuteDirectionBoth
	}

	if errHold := h.channelHandler.MuteOn(ctx, c.ChannelID, channel.MuteDirection(direction)); errHold != nil {
		log.Errorf("Could not mute the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not mute the channel")
	}

	return nil
}

// MuteOff moh off the call
func (h *callHandler) MuteOff(ctx context.Context, id uuid.UUID, direction call.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "MuteOff",
		"call_id":   id,
		"direction": direction,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if direction == call.MuteDirectionNone {
		direction = call.MuteDirectionBoth
	}

	if errHold := h.channelHandler.MuteOff(ctx, c.ChannelID, channel.MuteDirection(direction)); errHold != nil {
		log.Errorf("Could not mute off the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not mute off the channel")
	}

	return nil
}
