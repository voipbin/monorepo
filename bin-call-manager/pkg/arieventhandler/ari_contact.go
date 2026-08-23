package arieventhandler

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/common"
)

// EventHandlerContactStatusChange handles ContactStatusChange ARI event
func (h *eventHandler) EventHandlerContactStatusChange(ctx context.Context, evt interface{}) error {
	e := evt.(*ari.ContactStatusChange)

	log := logrus.WithFields(logrus.Fields{
		"func":  "EventHandlerContactStatusChange",
		"event": e,
	})
	log.Debugf("Received ContactStatusChange event: %v", e)

	ext, domain, err := common.ParseSIPURI(e.Endpoint.Resource)
	if err != nil {
		return fmt.Errorf("could not parse the endpoint")
	}

	// resolve the customer from the realm(domain)
	cd, err := h.reqHandler.RegistrarV1CustomerDomainGetByRealm(ctx, domain)
	if err != nil {
		// unknown realm. skip the contact refresh explicitly without failing
		// the event loop.
		log.Warnf("Could not get customer domain info. Skipping the contact refresh. domain: %s, err: %v", domain, err)
		return nil
	}

	// send refresh
	filters := map[string]any{
		"customer_id": cd.CustomerID,
		"extension":   ext,
	}
	if err := h.reqHandler.RegistrarV1ContactRefresh(ctx, filters); err != nil {
		log.Errorf("Could not handle the ContactStatusChange message. err: %v", err)
		return err
	}

	return nil
}
