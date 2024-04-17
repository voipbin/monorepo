package servicehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Conversation handles message receive for conversation
func (h *serviceHandler) Conversation(ctx context.Context, uri string, m []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Conversation",
		},
	)

	req := &hmhook.Hook{
		ReceviedURI:  uri,
		ReceivedData: m,
	}

	log.WithField("request", req).Debugf("Sending a hook message.")
	if err := h.reqHandler.ConversationV1Hook(ctx, req); err != nil {
		return err
	}

	return nil
}
