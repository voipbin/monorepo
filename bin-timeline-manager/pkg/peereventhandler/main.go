package peereventhandler

//go:generate mockgen -package peereventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/peerevent"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// PeerPair is peereventhandler's own primitive pair type — intentionally NOT
// dbhandler.PeerPairFilter (no dbhandler.* type leaks into this package's
// public interface) and NOT models/peerevent.PeerPair (the wire DTO belongs
// to the listenhandler/transport boundary, not the business logic layer).
type PeerPair struct {
	PeerType   string
	PeerTarget string
}

// PeerEventHandler interface for peer_events read operations.
type PeerEventHandler interface {
	List(ctx context.Context, customerID uuid.UUID, pairs []PeerPair, pageToken string, pageSize int) (*peerevent.PeerEventListResponse, error)
}

type peerEventHandler struct {
	db dbhandler.DBHandler
}

// NewPeerEventHandler creates a new PeerEventHandler.
func NewPeerEventHandler(db dbhandler.DBHandler) PeerEventHandler {
	return &peerEventHandler{
		db: db,
	}
}
