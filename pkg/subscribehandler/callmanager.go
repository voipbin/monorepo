package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmgroupdial "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMGroupdialCreated handles the call-manager's groupdial_created event.
func (h *subscribeHandler) processEventCMGroupdialCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupdialCreated",
		"event": m,
	})

	groupdial := &cmgroupdial.Groupdial{}
	if err := json.Unmarshal([]byte(m.Data), &groupdial); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupdialCreated(ctx, groupdial); errEvent != nil {
		log.Errorf("Could not handle the groupdial created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupdial created event.")
	}

	return nil
}

// processEventCMGroupdialAnswered handles the call-manager's groupdial_answered event.
func (h *subscribeHandler) processEventCMGroupdialAnswered(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupdialAnswered",
		"event": m,
	})

	groupdial := &cmgroupdial.Groupdial{}
	if err := json.Unmarshal([]byte(m.Data), &groupdial); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupdialAnswered(ctx, groupdial); errEvent != nil {
		log.Errorf("Could not handle the groupdial answered event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupdial answered event.")
	}

	return nil
}
