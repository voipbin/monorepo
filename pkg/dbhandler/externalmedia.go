package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

// ExternalMediaGet returns external media
func (h *handler) ExternalMediaGet(ctx context.Context, externalMediaID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	return h.cache.ExternalMediaGet(ctx, externalMediaID)
}

// ExternalMediaGetByReferenceID returns external media of the given reference id
func (h *handler) ExternalMediaGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*externalmedia.ExternalMedia, error) {
	return h.cache.ExternalMediaGetByReferenceID(ctx, referenceID)
}

// ExternalMediaSet sets external media.
func (h *handler) ExternalMediaSet(ctx context.Context, data *externalmedia.ExternalMedia) error {
	return h.cache.ExternalMediaSet(ctx, data)
}

// ExternalMediaDelete deletes external media.
func (h *handler) ExternalMediaDelete(ctx context.Context, externalMediaID uuid.UUID) error {
	return h.cache.ExternalMediaDelete(ctx, externalMediaID)
}
