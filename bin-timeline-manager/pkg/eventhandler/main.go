package eventhandler

//go:generate mockgen -package eventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// EventHandler interface for event operations.
type EventHandler interface {
	List(ctx context.Context, req *event.EventListRequest) (*event.EventListResponse, error)
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
