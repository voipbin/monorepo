package arieventhandler

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	ari "gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// EventHandlerStasisStart handles StasisStart ARI event
func (h *eventHandler) EventHandlerStasisStart(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisStart)

	log := log.WithFields(
		log.Fields{
			"channel_id":  e.Channel.ID,
			"asterisk_id": e.AsteriskID,
			"stasis_name": e.Application,
		})

	if !h.db.ChannelIsExist(e.Channel.ID, defaultExistTimeout) {
		log.Error("The given channel is not in our database.")
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseInterworking, 0)
		return fmt.Errorf("no channel found")
	}

	// get stasis name and stasis data
	stasisName := e.Application
	stasisData := make(map[string]string, 1)
	for k, v := range e.Args {
		stasisData[k] = v
	}

	// update data and stasis
	log.Debug("Updating channel stasis name and stasis data.")
	if err := h.db.ChannelSetStasisNameAndStasisData(ctx, e.Channel.ID, stasisName, stasisData); err != nil {
		// something went wrong. Hangup at here.
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated, 0)
		return err
	}

	cn, err := h.db.ChannelGet(ctx, e.Channel.ID)
	if err != nil {
		_ = h.reqHandler.AstChannelHangup(ctx, e.AsteriskID, e.Channel.ID, ari.ChannelCauseUnallocated, 0)
		return err
	}

	contextType := getContextType(stasisData["context"])
	switch contextType {
	case contextTypeCall:
		return h.callHandler.ARIStasisStart(ctx, cn, stasisData)

	case contextTypeConference:
		return h.confbridgeHandler.ARIStasisStart(ctx, cn, stasisData)

	default:
		log.Errorf("Could not find context type handler. context_type: %s", contextType)
		return fmt.Errorf("could not find context type handler. context_type: %s", contextType)
	}
}

// EventHandlerStasisEnd handles StasisEnd ARI event
func (h *eventHandler) EventHandlerStasisEnd(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.StasisEnd)

	if err := h.db.ChannelSetStasis(ctx, e.Channel.ID, ""); err != nil {
		// nothing we can do here
		return err
	}

	return nil
}
