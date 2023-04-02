package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// MOHOn moh the call
func (h *callHandler) MOHOn(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MOHOn",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.MOHOn(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not moh the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not moh the channel")
	}

	return nil
}

// MOHOff moh off the call
func (h *callHandler) MOHOff(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "MOHOff",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.MOHOff(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not moh off the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not moh off the channel")
	}

	return nil
}
