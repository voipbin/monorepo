package peereventhandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/peerevent"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// Pagination bounds for list-style queries, same clamp policy as
// eventhandler.List (pkg/eventhandler/event.go).
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

// List returns peer_events rows matching the given (peer_type, peer_target)
// pairs, scoped to customerID.
func (h *peerEventHandler) List(
	ctx context.Context,
	customerID uuid.UUID,
	pairs []PeerPair,
	pageToken string,
	pageSize int,
) (*peerevent.PeerEventListResponse, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "List",
		"customer_id": customerID,
	})

	if customerID == uuid.Nil {
		return nil, errors.New("customer_id is required")
	}
	if len(pairs) == 0 {
		return nil, errors.New("at least one peer_type+peer_target pair is required")
	}

	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	dbPairs := make([]dbhandler.PeerPairFilter, len(pairs))
	for i, p := range pairs {
		dbPairs[i] = dbhandler.PeerPairFilter{PeerType: p.PeerType, PeerTarget: p.PeerTarget}
	}

	rows, err := h.db.PeerEventList(ctx, customerID, dbPairs, pageToken, pageSize+1)
	if err != nil {
		log.Errorf("Could not list peer events. err: %v", err)
		return nil, errors.Wrap(err, "could not list peer events")
	}

	res := &peerevent.PeerEventListResponse{
		Result: rows,
	}

	if len(rows) > pageSize {
		res.Result = rows[:pageSize]
		res.NextPageToken = rows[pageSize-1].Timestamp.Format(commonutil.ISO8601Layout)
	}

	return res, nil
}
