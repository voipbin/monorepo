package arievent

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager/pkg/ari"
)

// eventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) eventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	log := log.WithFields(
		log.Fields{
			"channel":  e.Channel.ID,
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
		})

	if h.db.ChannelIsExist(e.Channel.ID, e.AsteriskID, defaultExistTimeout) == false {
		log.Error("The given channel is not in our database.")
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
		return fmt.Errorf("no channel found")
	}

	// set data and stasis
	data := make(map[string]interface{}, 1)
	for k, v := range e.Args {
		data[k] = v
	}
	stasis := e.Application

	// update data and stasis
	log.Debug("Updating channel stasis.")
	if err := h.db.ChannelSetDataAndStasis(ctx, e.AsteriskID, e.Channel.ID, data, stasis); err != nil {
		// something went wrong. Hangup at here.
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.AsteriskID, e.Channel.ID)
	if err != nil {
		h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	return h.callHandler.ARIStasisStart(cn)
}

// eventHandlerStasisEnd handles StasisEnd ARI event
func (h *eventHandler) eventHandlerStasisEnd(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisEnd)

	if err := h.db.ChannelSetStasis(ctx, e.AsteriskID, e.Channel.ID, ""); err != nil {
		// nothing we can do here
		return err
	}

	return nil
}
