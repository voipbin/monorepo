package eventhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/response"
)

// List returns events matching the request criteria.
func (h *eventHandler) List(ctx context.Context, req *request.V1DataEventsPost) (*response.V1DataEventsPost, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "List",
		"publisher":   req.Publisher,
		"resource_id": req.ResourceID,
		"events":      req.Events,
	})

	// Validate request
	if req.Publisher == "" {
		return nil, errors.New("publisher is required")
	}
	if req.ResourceID == uuid.Nil {
		return nil, errors.New("resource_id is required")
	}
	if len(req.Events) == 0 {
		return nil, errors.New("events filter is required")
	}

	// Apply defaults
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = request.DefaultPageSize
	}
	if pageSize > request.MaxPageSize {
		pageSize = request.MaxPageSize
	}

	// Query database (request pageSize + 1 to determine if more results exist)
	// Convert ServiceName to string for database query
	events, err := h.db.EventList(ctx, string(req.Publisher), req.ResourceID, req.Events, req.PageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list events. err: %v", err)
		return nil, errors.Wrap(err, "could not list events")
	}

	// Build response with pagination
	res := &response.V1DataEventsPost{
		Result: events,
	}

	// If we got more than pageSize, there are more results
	if len(events) > pageSize {
		res.Result = events[:pageSize]
		// Use timestamp of last returned event as next page token
		res.NextPageToken = events[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
	}

	return res, nil
}
