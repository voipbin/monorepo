package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMGroupcallCreated handles the call-manager's groupcall_created event.
func (h *subscribeHandler) processEventCMGroupcallCreated(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallCreated",
		"event": m,
	})

	groupcall := &cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &groupcall); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupcallCreated(ctx, groupcall); errEvent != nil {
		log.Errorf("Could not handle the groupcall created event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupcall created event.")
	}

	return nil
}

// processEventCMGroupcallProgressing handles the call-manager's groupcall_answered event.
func (h *subscribeHandler) processEventCMGroupcallProgressing(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMGroupcallAnswered",
		"event": m,
	})

	groupcall := &cmgroupcall.Groupcall{}
	if err := json.Unmarshal([]byte(m.Data), &groupcall); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if errEvent := h.agentHandler.EventGroupcallProgressing(ctx, groupcall); errEvent != nil {
		log.Errorf("Could not handle the groupcall answered event. err: %v", errEvent)
		return errors.Wrap(errEvent, "Could not handle the groupcall answered event.")
	}

	return nil
}
