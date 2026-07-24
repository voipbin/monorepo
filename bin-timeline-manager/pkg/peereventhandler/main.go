package peereventhandler

//go:generate mockgen -package peereventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-timeline-manager/models/peerevent"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
)

// PeerEventHandler interface for peer_events read operations.
//
// List takes commonaddress.Address values directly -- there is no
// peereventhandler-local pair type here. dbhandler.PeerEventList already
// accepts commonaddress.Address (only .Type/.Target are used internally
// for the search columns), so there is nothing left to translate at this
// layer; this mirrors eventhandler.EventHandler.List's "primitives only,
// no unnecessary wrapper type" shape while reusing the shared Address type
// (which, unlike dbhandler.PeerPairFilter previously, is meant to be a
// public, cross-package type).
type PeerEventHandler interface {
	List(ctx context.Context, customerID uuid.UUID, addrs []commonaddress.Address, pageToken string, pageSize int) (*peerevent.PeerEventListResponse, error)
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
