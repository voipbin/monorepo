package servicehandler

import (
	"context"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Message handles message receive
func (h *serviceHandler) Message(ctx context.Context, uri string, m []byte) error {
	req := &hmhook.Hook{
		ReceviedURI:  uri,
		ReceivedData: m,
	}

	if err := h.reqHandler.MessageV1Hook(ctx, req); err != nil {
		return err
	}

	return nil
}
