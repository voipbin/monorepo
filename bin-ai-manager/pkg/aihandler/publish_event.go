package aihandler

import (
	"context"

	"monorepo/bin-ai-manager/models/ai"
)

// publishAIEvent publishes an ai event with the engine key and the endpoint key (plaintext and
// ciphertext) removed. The event
// bus is archived by timeline-manager, so a customer's key must not ride on it.
// Only a copy is stripped; the caller's value keeps its key.
func (h *aiHandler) publishAIEvent(ctx context.Context, eventType string, a *ai.AI) {
	c := *a
	c.EngineKey = ""
	c.EngineBaseKey = ""
	c.EngineBaseKeyEnc = nil

	h.notifyHandler.PublishWebhookEvent(ctx, a.CustomerID, eventType, &c)
}
