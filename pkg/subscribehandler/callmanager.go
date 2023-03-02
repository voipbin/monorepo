package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupdial "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallProgressing handles the call-manager's call_progressing event.
func (h *subscribeHandler) processEventCMCallProgressing(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallProgressing",
		"event": m,
	})

	c := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	return h.agentHandler.AgentCallAnswered(ctx, c)
}

// processEventCMCallHangup handles the call-manager's call_hangup event.
func (h *subscribeHandler) processEventCMCallHangup(ctx context.Context, m *rabbitmqhandler.Event) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventCMCallHangup",
		"event": m,
	})

	c := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	return h.agentHandler.AgentCallHungup(ctx, c)
}

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
