package agenthandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

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

		// get agent info
		tmp, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		if tmp.Status != agent.StatusAvailable {
			// nothing to do.
			continue
		}

		// update agent status
		tmp, err = h.UpdateStatus(ctx, tmp.ID, agent.StatusRinging)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", tmp).Debugf("Updated agent status to the ringing. agent_id: %s", tmp.ID)
	}

	return nil
}

// EventGroupcallProgressing handles the call-manager's groupcall_progressing event
func (h *agentHandler) EventGroupcallProgressing(ctx context.Context, groupcall *cmgroupcall.Groupcall) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupcallProgressing",
		"groupcall": groupcall,
	})

	for _, destination := range groupcall.Destinations {
		if destination.Type != commonaddress.TypeAgent {
			// nothing to do
			continue
		}

		// parse agent id
		id := uuid.FromStringOrNil(destination.Target)
		if id == uuid.Nil {
			log.Errorf("Could not parse the agent id. target: %s", destination.Target)
			continue
		}

		// get agent info
		tmp, err := h.Get(ctx, id)
		if err != nil {
			log.Errorf("Could not get agent. err: %v", err)
			continue
		}

		// update agent status to busy
		tmp, err = h.UpdateStatus(ctx, tmp.ID, agent.StatusBusy)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			continue
		}
		log.WithField("agent", tmp).Debugf("Updated agent status to the busy. agent_id: %s", tmp.ID)
	}

	return nil
}
