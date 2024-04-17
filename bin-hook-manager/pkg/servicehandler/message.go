package servicehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	hmhook "monorepo/bin-hook-manager/models/hook"
)

// Message handles message receive
func (h *serviceHandler) Message(ctx context.Context, uri string, m []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "Message",
		},
	)

	req := &hmhook.Hook{
		ReceviedURI:  uri,
		ReceivedData: m,
	}

	log.WithField("request", req).Debugf("Created hook.")
	if err := h.reqHandler.MessageV1Hook(ctx, req); err != nil {
		return err
	}

	return nil
}
