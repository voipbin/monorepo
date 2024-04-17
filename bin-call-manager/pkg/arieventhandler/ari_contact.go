package arieventhandler

import (
	"context"
	"fmt"

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

	customerID, ext, err := common.ParseSIPURI(e.Endpoint.Resource)
	if err != nil {
		return fmt.Errorf("could not parse the endpoint")
	}

	// send refresh
	if err := h.reqHandler.RegistrarV1ContactRefresh(ctx, customerID, ext); err != nil {
		log.Errorf("Could not handle the ContactStatusChange message. err: %v", err)
		return err
	}

	return nil
}
