package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

// processEventActiveflowUpdated handles the activeflow deleted event.
func (h *subscribeHandler) processEventActiveflowUpdated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processEventActiveflowUpdated",
		"message": m,
	})

	a := &fmactiveflow.Activeflow{}
	if err := json.Unmarshal([]byte(m.Data), &a); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if a.Status != fmactiveflow.StatusEnded {
		// nothing to do
		return nil
	}

	if a.ReferenceType != fmactiveflow.ReferenceTypeCall {
		// nothing to do
		return nil
	}

	// safe to hanging up the hangup call
	_, _ = h.callHandler.HangingUp(ctx, a.ReferenceID, call.HangupReasonNormal)

	return nil
}
