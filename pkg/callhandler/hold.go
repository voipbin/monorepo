package callhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Hold holds the call
func (h *callHandler) HoldOn(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "HoldOn",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.HoldOn(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not hold the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not hold the channel")
	}

	return nil
}

// HoldOff unholds the call
func (h *callHandler) HoldOff(ctx context.Context, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "HoldOff",
		"call_id": id,
	})

	// get call info
	c, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get call info. err: %v", err)
		return errors.Wrap(err, "Could not get call info.")
	}

	if errHold := h.channelHandler.HoldOff(ctx, c.ChannelID); errHold != nil {
		log.Errorf("Could not unhold the channel. err: %v", errHold)
		return errors.Wrap(err, "Could not unhold the channel")
	}

	return nil
}
