package servicehandler

//go:generate mockgen -package servicehandler -destination ./mock_servicehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"
)

// ServiceHandler is interface for service handle
type ServiceHandler interface {
	Email(ctx context.Context, uri string, m []byte) error
	Message(ctx context.Context, uri string, m []byte) error
	Conversation(ctx context.Context, uri string, m []byte) error
}

type serviceHandler struct {
	reqHandler requesthandler.RequestHandler
}

// NewServiceHandler return ServiceHandler interface
func NewServiceHandler(reqHandler requesthandler.RequestHandler) ServiceHandler {
	return &serviceHandler{
		reqHandler: reqHandler,
	}
}
