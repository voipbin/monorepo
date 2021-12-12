package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferenceconfbridge"
)

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceConfbridgeSet(ctx context.Context, data *conferenceconfbridge.ConferenceConfbridge) error {

	return h.cache.ConferenceConfbridgeSet(ctx, data)
}

// ConferenceCreate creates a new conference record.
func (h *handler) ConferenceConfbridgeGet(ctx context.Context, confbridgeID uuid.UUID) (*conferenceconfbridge.ConferenceConfbridge, error) {

	return h.cache.ConferenceConfbridgeGet(ctx, confbridgeID)
}
