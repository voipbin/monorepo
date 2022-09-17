package arieventhandler

import (
	"context"

	log "github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
)

// EventHandlerContactStatusChange handles ContactStatusChange ARI event
func (h *eventHandler) EventHandlerContactStatusChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ContactStatusChange)

	log := log.WithFields(
		log.Fields{
			"asterisk": e.AsteriskID,
			"stasis":   e.Application,
			"aor":      e.ContactInfo.AOR,
		})

	// send update
	if err := h.reqHandler.RegistrarV1ContactUpdate(ctx, e.Endpoint.Resource); err != nil {
		log.Errorf("Could not handle the ContactStatusChange message. err: %v", err)
		return err
	}

	return nil
}
