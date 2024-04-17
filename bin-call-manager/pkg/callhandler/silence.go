package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SilenceOn silence the call
func (h *callHandler) SilenceOn(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "SilenceOn",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.SilenceOn(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not silence the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not silence the channel")
	}

	return nil
}

// SilenceOff moh off the call
func (h *callHandler) SilenceOff(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "SilenceOff",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.SilenceOff(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not silence off the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not silence off the channel")
	}

	return nil
}
