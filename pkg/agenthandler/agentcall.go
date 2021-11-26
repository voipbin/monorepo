package agenthandler

import (
	"context"

	"github.com/sirupsen/logrus"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
)

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
