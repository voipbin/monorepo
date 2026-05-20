package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"net/http"

	"monorepo/bin-common-handler/pkg/requesthandler"
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	Email(ctx context.Context, r *http.Request) error
	Message(ctx context.Context, r *http.Request) error
	Conversation(ctx context.Context, r *http.Request) (string, error)
	Billing(ctx context.Context, r *http.Request) error
}

type serviceHandler struct {
	reqHandler           requesthandler.RequestHandler
	paddleWebhookSecret string
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(reqHandler requesthandler.RequestHandler, paddleWebhookSecret string) ServiceHandler {
	return &serviceHandler{
		reqHandler:          reqHandler,
		paddleWebhookSecret: paddleWebhookSecret,
	}
}
