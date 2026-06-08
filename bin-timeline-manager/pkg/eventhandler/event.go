package eventhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonoutline "monorepo/bin-common-handler/models/outline"
	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/event"
)

// Pagination bounds for list-style queries. These are a business policy enforced
// by the event handler (the sole clamper), so they live here rather than in the
// transport request DTO package.
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

// List returns events matching the request criteria.
func (h *eventHandler) List(ctx context.Context, publisher commonoutline.ServiceName, resourceID uuid.UUID, events []string, pageToken string, pageSize int) (*event.EventListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "List",
		"publisher":   publisher,
		"resource_id": resourceID,
		"events":      events,
	})

	// Validate request
	if publisher == "" {
		return nil, errors.New("publisher is required")
	}
	if resourceID == uuid.Nil {
		return nil, errors.New("resource_id is required")
	}
	if len(events) == 0 {
		return nil, errors.New("events filter is required")
	}

	// Apply defaults
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Query database (request pageSize + 1 to determine if more results exist)
	// Convert ServiceName to string for database query
	rows, err := h.db.EventList(ctx, string(publisher), resourceID, events, pageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list events. err: %v", err)
		return nil, errors.Wrap(err, "could not list events")
	}

	// Build response with pagination
	res := &event.EventListResponse{
		Result: rows,
	}

	// If we got more than pageSize, there are more results
	if len(rows) > pageSize {
		res.Result = rows[:pageSize]
		// Use timestamp of last returned event as next page token
		res.NextPageToken = rows[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
	}

	return res, nil
}

// AggregatedList returns all events matching the given activeflow ID.
func (h *eventHandler) AggregatedList(ctx context.Context, activeflowID uuid.UUID, pageToken string, pageSize int) (*event.AggregatedEventListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "AggregatedList",
		"activeflow_id": activeflowID,
	})

	// Validate request
	if activeflowID == uuid.Nil {
		return nil, errors.New("activeflow_id is required")
	}

	// Apply defaults
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Query database (request pageSize + 1 to determine if more results exist)
	rows, err := h.db.AggregatedEventList(ctx, activeflowID.String(), pageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list aggregated events. err: %v", err)
		return nil, errors.Wrap(err, "could not list aggregated events")
	}

	// Build response with pagination
	res := &event.AggregatedEventListResponse{
		Result: rows,
	}

	// If we got more than pageSize, there are more results
	if len(rows) > pageSize {
		res.Result = rows[:pageSize]
		res.NextPageToken = rows[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
	}

	return res, nil
}
