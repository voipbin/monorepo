package servicehandler

import (
	"context"
	"fmt"
	"net/http"

	"monorepo/bin-api-manager/models/auth"

	"github.com/sirupsen/logrus"
)

// WebsockCreate validates the tag's ownership and returns the message info.
func (h *serviceHandler) WebsockCreate(ctx context.Context, a *auth.AuthIdentity, w http.ResponseWriter, r *http.Request) error {
	if a.IsDirect() {
		return fmt.Errorf("direct access not supported")
	}

	log := logrus.WithFields(logrus.Fields{
		"func":         "WebsockCreate",
		"display_name": a.DisplayName(),
	})

	if !a.IsAgent() || a.Agent == nil {
		log.Info("WebSocket requires agent authentication.")
		return fmt.Errorf("websocket requires agent authentication")
	}

	if errRun := h.websockHandler.RunSubscription(ctx, w, r, a); errRun != nil {
		log.Errorf("Could not run the websock handler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
