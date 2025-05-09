package dbhandler

import (
	"context"

	callapplication "monorepo/bin-call-manager/models/callapplication"
)

// CallApplicationAMDGet returns callapplication amd.
func (h *handler) CallApplicationAMDGet(ctx context.Context, channelID string) (*callapplication.AMD, error) {
	return h.cache.CallAppAMDGet(ctx, channelID)
}

// CallApplicationAMDSet sets callapplication amd.
func (h *handler) CallApplicationAMDSet(ctx context.Context, channelID string, app *callapplication.AMD) error {
	return h.cache.CallAppAMDSet(ctx, channelID, app)
}
