package dbhandler

import (
	"context"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupdial"
)

// GroupDialGet returns groupdial.
func (h *handler) GroupDialGet(ctx context.Context, id uuid.UUID) (*groupdial.GroupDial, error) {

	res, err := h.cache.GroupDialGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GroupDialCreate sets groupdial.
func (h *handler) GroupDialCreate(ctx context.Context, data *groupdial.GroupDial) error {

	return h.cache.GroupDialSet(ctx, data)
}
