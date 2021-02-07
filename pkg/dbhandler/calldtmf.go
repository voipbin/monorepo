package dbhandler

import (
	"context"

	uuid "github.com/gofrs/uuid"
)

// CallDTMFSet sets the call's received dtmf info
func (h *handler) CallDTMFSet(ctx context.Context, id uuid.UUID, dtmf string) error {
	return h.cache.CallDTMFSet(ctx, id, dtmf)
}

// CallDTMFGet gets the call's received dtmf info
func (h *handler) CallDTMFGet(ctx context.Context, id uuid.UUID) (string, error) {
	return h.cache.CallDTMFGet(ctx, id)
}

// CallDTMFReset resets the call's received dtmf info
func (h *handler) CallDTMFReset(ctx context.Context, id uuid.UUID) error {
	return h.cache.CallDTMFSet(ctx, id, "")
}
