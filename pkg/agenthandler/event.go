package agenthandler

import (
	"context"

	"github.com/pkg/errors"

	cmgroupdial "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// EventGroupdialCreated handles the call-manager's groupdial_created event
func (h *agentHandler) EventGroupdialCreated(ctx context.Context, groupdial *cmgroupdial.Groupdial) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupdialCreated",
		"groupdial": groupdial,
	})

	if groupdial.Destination.Type != commonaddress.TypeAgent {
		// nothing to do
		return nil
	}

	id := uuid.FromStringOrNil(groupdial.Destination.Target)
	if id == uuid.Nil {
		log.Errorf("Could not parse the agent id. target: %s", groupdial.Destination.Target)
		return nil
	}

	ag, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return errors.Wrap(err, "Could not get agent.")
	}

	if ag.Status == agent.StatusAvailable {
		ag, err = h.UpdateStatus(ctx, ag.ID, agent.StatusRinging)
		if err != nil {
			log.Errorf("Could not update agent status. err: %v", err)
			return errors.Wrap(err, "Could not update agent status.")
		}
		log.WithField("agent", ag).Debugf("Updated agent status to the ringing. agent_id: %s", ag.ID)
	}

	return nil
}

// EventGroupdialAnswered handles the call-manager's groupdial_answered event
func (h *agentHandler) EventGroupdialAnswered(ctx context.Context, groupdial *cmgroupdial.Groupdial) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "EventGroupdialAnswered",
		"groupdial": groupdial,
	})

	if groupdial.Destination.Type != commonaddress.TypeAgent {
		// nothing to do
		return nil
	}

	id := uuid.FromStringOrNil(groupdial.Destination.Target)
	if id == uuid.Nil {
		log.Errorf("Could not parse the agent id. target: %s", groupdial.Destination.Target)
		return nil
	}

	ag, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		return errors.Wrap(err, "Could not get agent.")
	}

	ag, err = h.UpdateStatus(ctx, ag.ID, agent.StatusBusy)
	if err != nil {
		log.Errorf("Could not update agent status. err: %v", err)
		return errors.Wrap(err, "Could not update agent status.")
	}
	log.WithField("agent", ag).Debugf("Updated agent status to the busy. agent_id: %s", ag.ID)

	return nil
}
