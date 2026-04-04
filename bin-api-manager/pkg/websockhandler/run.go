package websockhandler

import (
	"context"
	"net/http"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
)

func (h *websockHandler) RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error {
	return h.mediaStreamRun(ctx, w, r, referenceType, referenceID, encapsulation)
}

func (h *websockHandler) RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *auth.AuthIdentity) error {
	return h.subscriptionRun(ctx, w, r, a)
}
