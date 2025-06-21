package subscribehandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-common-handler/models/sock"
	smpod "monorepo/bin-sentinel-manager/models/pod"

	"github.com/sirupsen/logrus"
)

// processEventSMPodDeleted handles the sentinel-manager's pod_deleted event.
func (h *subscribeHandler) processEventSMPodDeleted(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEventSMPodDeleted",
		"message": m,
	})
	log.Debugf("Executing the event handler.")

	p := &smpod.Pod{}
	if err := json.Unmarshal([]byte(m.Data), &p); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.callHandler.EventSMPodDeleted(ctx, p); errEvent != nil {
		log.Errorf("Could not handle the event correctly. The call handler returned an error. err: %v", errEvent)
	}

	return nil
}
