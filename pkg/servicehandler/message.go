package servicehandler

import (
	"context"

	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/hook-manager.git/models/hook"
)

// Message handles message receive
func (h *serviceHandler) Message(ctx context.Context, uri string, m []byte) error {
	log := logrus.WithFields(
		logrus.Fields{
			"func": "HookMessage",
		},
	)

	req := &hook.Hook{
		ReceviedURI:  uri,
		ReceivedData: m,
	}
	log.WithField("request", req).Debugf("Created hook.")

	return nil
}
