package websockhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package websockhandler -destination ./mock_hookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
)

// WebsockHandler defines
type WebsockHandler interface {
	Run(ctx context.Context, w http.ResponseWriter, r *http.Request, a *amagent.Agent) error
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type websockHandler struct {
}

// NewWebsockHandler creates a new HookHandler
func NewWebsockHandler() WebsockHandler {

	res := &websockHandler{}

	return res
}
