package websockhandler

//go:generate mockgen -package websockhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"monorepo/bin-api-manager/pkg/streamhandler"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-api-manager/models/auth"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

// WebsockHandler defines
type WebsockHandler interface {
	RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *auth.AuthIdentity) error
	RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type websockHandler struct {
	reqHandler    requesthandler.RequestHandler
	streamHandler streamhandler.StreamHandler
	scopeRefCount *scopeRefCount // shared across all connections on this pod
}

// NewWebsockHandler creates a new HookHandler
func NewWebsockHandler(
	reqHandler requesthandler.RequestHandler,
	streamHandler streamhandler.StreamHandler,
	sockHandler sockhandler.SockHandler,
	queueNamePod string,
) WebsockHandler {

	res := &websockHandler{
		reqHandler:    reqHandler,
		streamHandler: streamHandler,
		scopeRefCount: newScopeRefCount(sockHandler, queueNamePod, string(commonoutline.QueueNameWebhookEventTopic)),
	}

	endpointInit()

	return res
}
