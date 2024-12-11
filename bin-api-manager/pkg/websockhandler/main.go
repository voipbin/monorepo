package websockhandler

//go:generate mockgen -package websockhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"monorepo/bin-api-manager/pkg/streamhandler"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"monorepo/bin-common-handler/pkg/requesthandler"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

// WebsockHandler defines
type WebsockHandler interface {
	RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error
	RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType cmexternalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type websockHandler struct {
	reqHandler    requesthandler.RequestHandler
	streamHandler streamhandler.StreamHandler
}

// NewWebsockHandler creates a new HookHandler
func NewWebsockHandler(reqHandler requesthandler.RequestHandler, streamHandler streamhandler.StreamHandler) WebsockHandler {

	res := &websockHandler{
		reqHandler:    reqHandler,
		streamHandler: streamHandler,
	}

	endpointInit()

	return res
}
