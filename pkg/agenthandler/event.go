package agenthandler

import (
	"context"

	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// EventGroupcallCreated handles the call-manager's groupcall_created event
func (h *agentHandler) EventGroupcallCreated(ctx context.Context, groupcall *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallCreated",
		"groupcall": groupcall,
	})

	for _, destination := range groupcall.Destinations {
		if destination.Type != commonaddress.TypeAgent {
			// nothing to do
			continue
		}

		id := uuid.FromStringOrNil(destination.Target)
		if id == uuid.Nil {
			log.Errorf("Could not parse the agent id. target: %s", destination.Target)
			continue
		}

		ag, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		if ag.Status != agent.StatusAvailable {
			// nothing to do.
			continue
		}

		ag, err = h.UpdateStatus(ctx, ag.ID, agent.StatusRinging)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", ag).Debugf("Updated agent status to the ringing. agent_id: %s", ag.ID)
	}

	return nil
}

// EventGroupcallAnswered handles the call-manager's groupcall_answered event
func (h *agentHandler) EventGroupcallAnswered(ctx context.Context, groupcall *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupdialAnswered",
		"groupcall": groupcall,
	})

	for _, destination := range groupcall.Destinations {
		if destination.Type != commonaddress.TypeAgent {
			// nothing to do
			continue
		}

		id := uuid.FromStringOrNil(destination.Target)
		if id == uuid.Nil {
			log.Errorf("Could not parse the agent id. target: %s", destination.Target)
			continue
		}

		ag, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		ag, err = h.UpdateStatus(ctx, ag.ID, agent.StatusBusy)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", ag).Debugf("Updated agent status to the busy. agent_id: %s", ag.ID)
	}

	return nil
}
