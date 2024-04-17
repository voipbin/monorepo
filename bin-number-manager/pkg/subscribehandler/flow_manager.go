package subscribehandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	fmflow "monorepo/bin-flow-manager/models/flow"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// processEventFMFlowDeleted handles the flow-manager's flow_deleted event.
func (h *subscribeHandler) processEventFMFlowDeleted(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventFMFlowDeleted",
		"event": m,
	})

	f := &fmflow.Flow{}
	if err := json.Unmarshal([]byte(m.Data), &f); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	log.WithField("flow", f).Debugf("Received flow deleted event. flow_id: %s", f.ID)
	if errRemove := h.numberHandler.EventFlowDeleted(ctx, f); errRemove != nil {
		log.Errorf("Could not handle the flow deleted event. err: %v", errRemove)
		return errors.Wrap(errRemove, "Could not handle the flow deleted event.")
	}

	return nil
}
