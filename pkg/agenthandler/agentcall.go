package agenthandler

import (
	"context"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// AgentCallAnswered handles the situation for agent's call answered.
func (h *agentHandler) AgentCallAnswered(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AgentCallAnswered",
		"call_id": c.ID,
	})
	log.Debug("The agent's call has answered.")

	// get agent call
	ac, err := h.db.AgentCallGet(ctx, c.ID)
	if err != nil {
		// not an agent call. ignore it.
		return nil
	}

	// set agent's status to busy
	if err := h.db.AgentSetStatus(ctx, ac.AgentID, agent.StatusBusy); err != nil {
		log.Errorf("Could not update agent's status. err: %v", err)

		// we couldn't update the agent's status.
		// but the agent answered the call already, we just keep going.
	}

	// get agent dial
	ad, err := h.db.AgentDialGet(ctx, ac.AgentID)
	if err != nil {
		log.Errorf("Could not get agent dial. err: %v", err)
		return err
	}

	// hang up the other agent calls.
	for _, callID := range ad.CallIDs {
		if callID == c.ID {
			continue
		}

		tmpCall, err := h.reqHandler.CMV1CallHangup(ctx, callID)
		if err != nil {
			log.Errorf("Could not hangup the call. err: %v", err)
			continue
		}
		log.WithField("call", tmpCall).Debug("Hangup the call.")
	}

	return nil
}

// AgentCallHungup handles the situation for agent's call hungup.
func (h *agentHandler) AgentCallHungup(ctx context.Context, c *cmcall.Call) error {
	log := logrus.WithFields(logrus.Fields{
		"func":    "AgentCallHungup",
		"call_id": c.ID,
	})
	log.Debug("The agent's call has answered.")

	// currently, we don't do anything here.
	// the plan was, if the agent's call has hungup put the agent's status to the available.
	// but if the agent's call is more than 2, it is hard to know this is right time or not.
	// so leave it as is.
	return nil
}
