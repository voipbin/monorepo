package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

// Answer answers the given channel's playback
func (h *channelHandler) Answer(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Answer",
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

	if errAnswer := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, cn.ID); errAnswer != nil {
		log.Errorf("Could not answer. err: %v", errAnswer)
		return errors.Wrap(errAnswer, "could not answer")
	}

	return nil
}

// DTMFSend sends the dtmfs to the given channel
func (h *channelHandler) DTMFSend(ctx context.Context, id string, digit string, duration int, before int, between int, after int) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "DTMFSend",
		"channel_id": id,
		"duration":   duration,
		"before":     before,
		"between":    between,
		"after":      after,
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

	if errDTMF := h.reqHandler.AstChannelDTMF(ctx, cn.AsteriskID, cn.ID, digit, duration, before, between, after); errDTMF != nil {
		log.Errorf("Could not send the dtmfs. err: %v", errDTMF)
		return errors.Wrap(errDTMF, "could not send the dtmfs")
	}

	return nil
}

// Record starts the channel recording
func (h *channelHandler) Record(ctx context.Context, id string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Record",
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

	if errRecord := h.reqHandler.AstChannelRecord(ctx, cn.AsteriskID, cn.ID, filename, format, duration, silence, beep, endKey, ifExists); errRecord != nil {
		log.Errorf("Could not send the dtmfs. err: %v", errRecord)
		return errors.Wrap(errRecord, "could not record the channel")
	}

	return nil
}

// Dial dials the channel
func (h *channelHandler) Dial(ctx context.Context, id string, caller string, timeout int) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Dial",
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

	if errRecord := h.reqHandler.AstChannelDial(ctx, cn.AsteriskID, cn.ID, caller, timeout); errRecord != nil {
		log.Errorf("Could not dial the channel. err: %v", errRecord)
		return errors.Wrap(errRecord, "could not dial the channel")
	}

	return nil
}

// Redirect redirects the given channel to the given context.exten.priority.
func (h *channelHandler) Redirect(ctx context.Context, id string, contextName string, exten string, priority string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Redirect",
		"channel_id": id,
		"context":    contextName,
		"exten":      exten,
		"priority":   priority,
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

	if err := h.reqHandler.AstAMIRedirect(ctx, cn.AsteriskID, cn.ID, contextName, exten, priority); err != nil {
		log.Errorf("Could not redirect the channel.err: %v", err)
		return errors.Wrap(err, "could not redirect the channel")
	}

	return nil
}

// Continue continues the given channel to the given context.exten,priority.label
func (h *channelHandler) Continue(ctx context.Context, id string, context string, exten string, priority int, label string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Continue",
		"channel_id": id,
		"context":    context,
		"exten":      exten,
		"priority":   priority,
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

	if err := h.reqHandler.AstChannelContinue(ctx, cn.AsteriskID, cn.ID, context, exten, priority, label); err != nil {
		log.Errorf("Could not continue the channel.err: %v", err)
		return errors.Wrap(err, "could not continue the channel")
	}

	return nil
}

// Ring rings the channel
func (h *channelHandler) Ring(ctx context.Context, id string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "Ring",
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

	if errRing := h.reqHandler.AstChannelRing(ctx, cn.AsteriskID, cn.ID); errRing != nil {
		log.Errorf("Could not ring. err: %v", errRing)
		return errors.Wrap(errRing, "could not ring")
	}

	return nil
}
