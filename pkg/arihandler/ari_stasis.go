package arihandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
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

	if !h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) {
		log.Error("The given channel is not in our database.")
		_ = h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking)
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
	if err := h.db.ChannelSetStasis(ctx, e.Channel.ID, stasis); err != nil {
		// something went wrong. Hangup at here.
		_ = h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		_ = h.reqHandler.AstChannelHangup(e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated)
		return err
	}

	contextType := getContextType(data["context"])
	switch contextType {
	case contextTypeCall:
		return h.callHandler.ARIStasisStart(cn, data)

	case contextTypeConference:
		return h.confHandler.ARIStasisStart(cn, data)

	default:
		log.Errorf("Could not find context type handler. context_type: %s", contextType)
		return fmt.Errorf("could not find context type handler. context_type: %s", contextType)
	}
}

// eventHandlerStasisEnd handles StasisEnd ARI event
func (h *eventHandler) eventHandlerStasisEnd(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisEnd)

	if err := h.db.ChannelSetStasis(ctx, e.Channel.ID, ""); err != nil {
		// nothing we can do here
		return err
	}

	return nil
}
