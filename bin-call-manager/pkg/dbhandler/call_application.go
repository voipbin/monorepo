package dbhandler

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

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

// CallRecoveryClaim takes the recovery claim for the given call. Cache-only data, passed through like the AMD info.
func (h *handler) CallRecoveryClaim(ctx context.Context, id uuid.UUID, ttl time.Duration) (bool, error) {
	return h.cache.CallRecoveryClaim(ctx, id, ttl)
}

// ConfbridgeParticipantClaim takes the add-participant claim for the given confbridge. Cache-only data, passed through like the recovery claim.
func (h *handler) ConfbridgeParticipantClaim(ctx context.Context, confbridgeID uuid.UUID, ttl time.Duration) (bool, error) {
	return h.cache.ConfbridgeParticipantClaim(ctx, confbridgeID, ttl)
}

// ConfbridgeParticipantRelease releases the add-participant claim of the given confbridge.
func (h *handler) ConfbridgeParticipantRelease(ctx context.Context, confbridgeID uuid.UUID) error {
	return h.cache.ConfbridgeParticipantRelease(ctx, confbridgeID)
}
