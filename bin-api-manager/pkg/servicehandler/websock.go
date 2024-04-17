package servicehandler

import (
	"context"
	"net/http"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/sirupsen/logrus"
)

// WebsockCreate validates the tag's ownership and returns the message info.
func (h *serviceHandler) WebsockCreate(ctx context.Context, a *amagent.Agent, w http.ResponseWriter, r *http.Request) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "WebsockCreate",
		"agent": a,
	})

	if errRun := h.websockHandler.RunSubscription(ctx, w, r, a); errRun != nil {
		log.Errorf("Could not run the websock handler correctly. err: %v", errRun)
		return errRun
	}

	return nil
}
