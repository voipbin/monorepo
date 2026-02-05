package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SilenceOn silences the given channel
func (h *channelHandler) SilenceOn(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SilenceOn",
		"channel_id": id,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete != nil {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if errReq := h.reqHandler.AstChannelSilenceOn(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not silence the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not silence the channel")
	}

	return nil
}

// SilenceOff unsilences the given channel
func (h *channelHandler) SilenceOff(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "SilenceOff",
		"channel_id": id,
	})

	cn, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get channel info. err: %v", err)
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete != nil {
		log.Errorf("The channel has hungup already.")
		return fmt.Errorf("the channel has hungup already")
	}

	if errReq := h.reqHandler.AstChannelSilenceOff(ctx, cn.AsteriskID, cn.ID); errReq != nil {
		log.Errorf("Could not unsilence the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not unsilence the channel")
	}

	return nil
}
