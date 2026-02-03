package eventhandler

//go:generate mockgen -package eventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// EventHandler interface for event operations.
type EventHandler interface {
	List(ctx context.Context, req *request.V1DataEventsPost) (*response.V1DataEventsPost, error)
}

type eventHandler struct {
	db dbhandler.DBHandler
}

// NewEventHandler creates a new EventHandler.
func NewEventHandler(db dbhandler.DBHandler) EventHandler {
	return &eventHandler{
		db: db,
	}
}
