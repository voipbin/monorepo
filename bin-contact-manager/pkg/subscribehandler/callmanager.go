package subscribehandler

import (
	"context"
	"encoding/json"

	callmodel "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/models/sock"
)

// processEventCallManagerCallCreated handles the call-manager's call_created event
// and projects it into the CRM interaction timeline.
func (h *subscribeHandler) processEventCallManagerCallCreated(ctx context.Context, m *sock.Event) error {
	var payload callmodel.WebhookMessage
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		return err
	}

	return h.contactHandler.EventCallCreated(ctx, &payload)
}
