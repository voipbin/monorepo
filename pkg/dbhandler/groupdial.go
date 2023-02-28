package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
)

// GroupdialGet returns groupdial.
func (h *handler) GroupdialGet(ctx context.Context, id uuid.UUID) (*groupdial.Groupdial, error) {

	res, err := h.cache.GroupdialGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GroupdialCreate sets groupdial.
func (h *handler) GroupdialCreate(ctx context.Context, data *groupdial.Groupdial) error {

	return h.cache.GroupdialSet(ctx, data)
}

// GroupdialUpdate updates the groupdial.
func (h *handler) GroupdialUpdate(ctx context.Context, data *groupdial.Groupdial) error {

	return h.cache.GroupdialSet(ctx, data)
}
