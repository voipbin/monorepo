package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

// MuteOn mutes the given channel
func (h *channelHandler) MuteOn(ctx context.Context, id string, direction channel.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "MuteOn",
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

	if errReq := h.reqHandler.AstChannelMuteOn(ctx, cn.AsteriskID, cn.ID, string(direction)); errReq != nil {
		log.Errorf("Could not mute the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not mute the channel")
	}

	_, err = h.UpdateMuteDirection(ctx, cn.ID, direction)
	if err != nil {
		log.Errorf("Could not update the mute direction. err: %v", err)
	}

	return nil
}

// MuteOff unmutes the given channel
func (h *channelHandler) MuteOff(ctx context.Context, id string, muteDirection channel.MuteDirection) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "MuteOff",
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

	if errReq := h.reqHandler.AstChannelMuteOff(ctx, cn.AsteriskID, cn.ID, string(muteDirection)); errReq != nil {
		log.Errorf("Could not unmute the channel. err: %v", errReq)
		return errors.Wrap(errReq, "could not unmute the channel")
	}

	newDirection := channel.MuteDirectionNone
	if muteDirection == channel.MuteDirectionBoth {
		newDirection = channel.MuteDirectionNone
	} else if cn.MuteDirection != muteDirection {
		switch cn.MuteDirection {
		case channel.MuteDirectionBoth:
			if muteDirection == channel.MuteDirectionIn {
				newDirection = channel.MuteDirectionOut
			} else {
				newDirection = channel.MuteDirectionIn
			}

		default:
			newDirection = cn.MuteDirection
		}
	}

	_, err = h.UpdateMuteDirection(ctx, cn.ID, newDirection)
	if err != nil {
		log.Errorf("Could not update the mute direction. err: %v", err)
	}

	return nil
}
