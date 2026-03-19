package servicehandler

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Conversation handles message receive for conversation
func (h *serviceHandler) Conversation(ctx context.Context, r *http.Request) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Conversation",
		},
	)

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("could not read request body: %w", err)
	}

	req := &hmhook.Hook{
		ReceviedURI:  r.Host + r.URL.Path,
		ReceivedData: data,
	}

	log.WithField("request", req).Debugf("Sending a hook message.")
	if err := h.reqHandler.ConversationV1Hook(ctx, req); err != nil {
		return fmt.Errorf("could not send the hook: %w", err)
	}

	return nil
}
