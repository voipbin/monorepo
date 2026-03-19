package servicehandler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Message handles message receive
func (h *serviceHandler) Message(ctx context.Context, r *http.Request) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	req := &hmhook.Hook{
		ReceviedURI:  r.Host + r.URL.Path,
		ReceivedData: data,
	}

	if err := h.reqHandler.MessageV1Hook(ctx, req); err != nil {
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}
