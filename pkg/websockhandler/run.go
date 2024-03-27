package websockhandler

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cmexternalmedia "gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
)

func (h *websockHandler) RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error {
	return h.mediaStreamRun(ctx, w, r, referenceType, referenceID, encapsulation)
}

func (h *websockHandler) RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error {
	return h.subscriptionRun(ctx, w, r, a)
}
