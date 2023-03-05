package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// GroupcallGet returns groupcall.
func (h *handler) GroupcallGet(ctx context.Context, id uuid.UUID) (*groupcall.Groupcall, error) {

	res, err := h.cache.GroupcallGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GroupcallCreate sets groupcall.
func (h *handler) GroupcallCreate(ctx context.Context, data *groupcall.Groupcall) error {

	data.TMCreate = h.utilHandler.GetCurTime()
	data.TMUpdate = DefaultTimeStamp
	data.TMDelete = DefaultTimeStamp

	return h.cache.GroupcallSet(ctx, data)
}

// GroupcallUpdate updates the groupcall.
func (h *handler) GroupcallUpdate(ctx context.Context, data *groupcall.Groupcall) error {

	data.TMUpdate = h.utilHandler.GetCurTime()

	return h.cache.GroupcallSet(ctx, data)
}
