package subscribehandler

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

// processEventCMCallAnswered handles the call-manager's call_answered event.
func (h *subscribeHandler) processEventCMCallAnswered(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	c := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	return h.agentHandler.AgentCallAnswered(ctx, c)
}

// processEventCMCallHungup handles the call-manager's call_hungup event.
func (h *subscribeHandler) processEventCMCallHungup(m *rabbitmqhandler.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"event": m,
		},
	)
	log.Debugf("Received call event. event: %s", m.Type)

	c := &cmcall.Call{}
	if err := json.Unmarshal([]byte(m.Data), &c); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	return h.agentHandler.AgentCallHungup(ctx, c)
}
