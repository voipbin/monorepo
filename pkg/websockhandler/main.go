package websockhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package websockhandler -destination ./mock_hookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
)

// WebsockHandler defines
type WebsockHandler interface {
	Run(ctx context.Context, w http.ResponseWriter, r *http.Request, customerID uuid.UUID) error
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
