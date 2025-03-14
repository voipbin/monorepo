package servicehandler

import (
	"context"
	hmhook "monorepo/bin-hook-manager/models/hook"

	"github.com/pkg/errors"
)

// Email handles email receive
func (h *serviceHandler) Email(ctx context.Context, uri string, m []byte) error {
	req := &hmhook.Hook{
		ReceviedURI:  uri,
		ReceivedData: m,
	}

	if errHook := h.reqHandler.EmailV1Hooks(ctx, req); errHook != nil {
		return errors.Wrapf(errHook, "could not send the hook")
	}

	return nil
}
