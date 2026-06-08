package eventhandler

//go:generate mockgen -package eventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/correlation"
	"monorepo/bin-timeline-manager/models/event"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
)

// EventHandler interface for event operations.
type EventHandler interface {
	List(ctx context.Context, req *request.V1DataEventsPost) (*event.EventListResponse, error)
	AggregatedList(ctx context.Context, req *request.V1DataAggregatedEventsPost) (*event.AggregatedEventListResponse, error)
	ResourceCorrelationGet(ctx context.Context, resourceID uuid.UUID) (*correlation.ResourceCorrelation, error)
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
