package arieventhandler

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
)

// EventHandlerContactStatusChange handles ContactStatusChange ARI event
func (h *eventHandler) EventHandlerContactStatusChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ContactStatusChange)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerContactStatusChange",
		"event": e,
	})

	endpoint := strings.TrimSuffix(e.Endpoint.Resource, common.DomainSIPSuffix)
	log.Debugf("Updating contact info. endpoint: %s", endpoint)

	// send update
	if err := h.reqHandler.RegistrarV1ContactUpdate(ctx, endpoint); err != nil {
		log.Errorf("Could not handle the ContactStatusChange message. err: %v", err)
		return err
	}

	return nil
}
