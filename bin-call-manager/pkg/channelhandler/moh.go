package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

// MOHOn moh the given channel
func (h *channelHandler) MOHOn(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "MOHOn",
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

	if errReq := h.reqHandler.AstChannelMusicOnHoldOn(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not moh the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not moh the channel")
	}

	return nil
}

// MOHOff moh off the given channel
func (h *channelHandler) MOHOff(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "MOHOff",
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

	if errReq := h.reqHandler.AstChannelMusicOnHoldOff(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not moh off the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not moh off the channel")
	}

	return nil
}
