package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// ExternalMediaGet returns external media
func (h *handler) ExternalMediaGet(ctx context.Context, callID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	return h.cache.CallExternalMediaGet(ctx, callID)
}

// ExternalMediaSet sets external media.
func (h *handler) ExternalMediaSet(ctx context.Context, callID uuid.UUID, data *externalmedia.ExternalMedia) error {
	return h.cache.CallExternalMediaSet(ctx, callID, data)
}

// ExternalMediaDelete deletes external media.
func (h *handler) ExternalMediaDelete(ctx context.Context, callID uuid.UUID) error {
	return h.cache.CallExternalMediaDelete(ctx, callID)
}
