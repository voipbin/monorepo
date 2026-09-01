package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/sock"
	smcontainer "monorepo/bin-sentinel-manager/models/container"

	"github.com/sirupsen/logrus"
)

// processEventSMContainerDied handles the sentinel-manager's container_died event.
//
// It replaces processEventSMPodDeleted (VOIP-1418): the payload is no longer a raw Kubernetes Pod
// but sentinel's own minimal container.Event.
func (h *subscribeHandler) processEventSMContainerDied(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEventSMContainerDied",
		"message": m,
	})
	log.Debugf("Executing the event handler.")

	c := &smcontainer.Event{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.callHandler.EventSMContainerDied(ctx, c); errEvent != nil {
		log.Errorf("Could not handle the event correctly. The call handler returned an error. err: %v", errEvent)
	}

	return nil
}
