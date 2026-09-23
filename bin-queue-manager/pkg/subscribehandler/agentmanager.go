package subscribehandler

import (
	"context"
	"encoding/json"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-common-handler/models/sock"

	"github.com/sirupsen/logrus"
)

// processEventAMAgentStatusUpdated handles the agent-manager's
// agent_status_updated event (VOIP-1539 §3.5, event entry point B): only
// status==available is actionable here, so anything else is a no-op --
// forwarding this decision to queuecallHandler keeps the filter next to the
// match() call it gates, instead of splitting the "available means matchable"
// rule across two packages.
func (h *subscribeHandler) processEventAMAgentStatusUpdated(ctx context.Context, m *sock.Event) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func":  "processEventAMAgentStatusUpdated",
			"event": m,
		},
	)

	e := amagent.Agent{}
	if err := json.Unmarshal([]byte(m.Data), &e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	if e.Status != amagent.StatusAvailable {
		return nil
	}

	h.queuecallHandler.EventAMAgentAvailable(ctx, e)

	return nil
}
