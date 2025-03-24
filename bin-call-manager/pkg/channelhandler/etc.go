package channelhandler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"monorepo/bin-call-manager/pkg/dbhandler"
)

// Answer answers the given channel's playback
func (h *channelHandler) Answer(ctx context.Context, id string) error {
	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if errAnswer := h.reqHandler.AstChannelAnswer(ctx, cn.AsteriskID, cn.ID); errAnswer != nil {
		return errors.Wrap(errAnswer, "could not answer")
	}

	return nil
}

// DTMFSend sends the dtmfs to the given channel
func (h *channelHandler) DTMFSend(ctx context.Context, id string, digit string, duration int, before int, between int, after int) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if errDTMF := h.reqHandler.AstChannelDTMF(ctx, cn.AsteriskID, cn.ID, digit, duration, before, between, after); errDTMF != nil {
		return errors.Wrap(errDTMF, "could not send the dtmfs")
	}

	return nil
}

// Record starts the channel recording
func (h *channelHandler) Record(ctx context.Context, id string, filename string, format string, duration int, silence int, beep bool, endKey string, ifExists string) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if errRecord := h.reqHandler.AstChannelRecord(ctx, cn.AsteriskID, cn.ID, filename, format, duration, silence, beep, endKey, ifExists); errRecord != nil {
		return errors.Wrap(errRecord, "could not record the channel")
	}

	return nil
}

// Dial dials the channel
func (h *channelHandler) Dial(ctx context.Context, id string, caller string, timeout int) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if errRecord := h.reqHandler.AstChannelDial(ctx, cn.AsteriskID, cn.ID, caller, timeout); errRecord != nil {
		return errors.Wrap(errRecord, "could not dial the channel")
	}

	return nil
}

// Redirect redirects the given channel to the given context.exten.priority.
func (h *channelHandler) Redirect(ctx context.Context, id string, contextName string, exten string, priority string) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if err := h.reqHandler.AstAMIRedirect(ctx, cn.AsteriskID, cn.ID, contextName, exten, priority); err != nil {
		return errors.Wrap(err, "could not redirect the channel")
	}

	return nil
}

// Continue continues the given channel to the given context.exten,priority.label
func (h *channelHandler) Continue(ctx context.Context, id string, context string, exten string, priority int, label string) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if err := h.reqHandler.AstChannelContinue(ctx, cn.AsteriskID, cn.ID, context, exten, priority, label); err != nil {
		return errors.Wrap(err, "could not continue the channel")
	}

	return nil
}

// Ring rings the channel
func (h *channelHandler) Ring(ctx context.Context, id string) error {

	cn, err := h.Get(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get channel info")
	}

	if cn.TMDelete < dbhandler.DefaultTimeStamp {
		return fmt.Errorf("the channel has hungup already")
	}

	if errRing := h.reqHandler.AstChannelRing(ctx, cn.AsteriskID, cn.ID); errRing != nil {
		return errors.Wrap(errRing, "could not ring")
	}

	return nil
}
