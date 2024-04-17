package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

// HoldOn holds the given channel
func (h *channelHandler) HoldOn(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "HoldOn",
		"channel_id": id,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if errReq := h.reqHandler.AstChannelHoldOn(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not hold the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not hold the channel")
	}

	return nil
}

// HoldOff unholds the given channel
func (h *channelHandler) HoldOff(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "HoldOff",
		"channel_id": id,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if errReq := h.reqHandler.AstChannelHoldOff(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not unhold the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not unhold the channel")
	}

	return nil
}
