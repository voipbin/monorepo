package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
)

// processEventFMActiveflowUpdated handles the activeflow updated event.
func (h *subscribeHandler) processEventFMActiveflowUpdated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEventFMActiveflowUpdated",
		"message": m,
	})
	log.Debugf("Executing the event handler.")

	a := &fmactiveflow.Activeflow{}
	if err := json.Unmarshal([]byte(m.Data), &a); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errCall := h.callHandler.EventFMActiveflowUpdated(ctx, a); errCall != nil {
		log.Errorf("Could not handle the event correctly. The call handler returned an error. err: %v", errCall)
	}

	return nil
}
