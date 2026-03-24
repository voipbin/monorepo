package directhandler

import (
	"context"

	"monorepo/bin-direct-manager/models/direct"
)

// publishEvent publishes a direct event
func (h *directHandler) publishEvent(ctx context.Context, eventType string, d *direct.Direct) {
	h.notifyhandler.PublishEvent(ctx, eventType, d)
}
