package outboundconfighandler

//go:generate mockgen -package outboundconfighandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// OutboundConfigHandler manages OutboundConfig resources.
type OutboundConfigHandler interface {
	Delete(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error)
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*outboundconfig.OutboundConfig, error)
	GetByID(ctx context.Context, id uuid.UUID) (*outboundconfig.OutboundConfig, error)
	List(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*outboundconfig.OutboundConfig, error)
	Create(ctx context.Context, customerID uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error)
	Update(ctx context.Context, id uuid.UUID, req *outboundconfig.UpdateRequest) (*outboundconfig.OutboundConfig, error)
}

type outboundConfigHandler struct {
	utilHandler  utilhandler.UtilHandler
	db           dbhandler.DBHandler
	cacheHandler cachehandler.CacheHandler
	reqHandler   requesthandler.RequestHandler
}

// NewOutboundConfigHandler creates an OutboundConfigHandler.
func NewOutboundConfigHandler(
	utilHandler utilhandler.UtilHandler,
	db dbhandler.DBHandler,
	cacheHandler cachehandler.CacheHandler,
	reqHandler requesthandler.RequestHandler,
) OutboundConfigHandler {
	return &outboundConfigHandler{
		utilHandler:  utilHandler,
		db:           db,
		cacheHandler: cacheHandler,
		reqHandler:   reqHandler,
	}
}
