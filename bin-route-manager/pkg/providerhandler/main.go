package providerhandler

//go:generate mockgen -package providerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

type providerHandler struct {
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// ProviderHandler interface
type ProviderHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*provider.Provider, error)
	Create(
		ctx context.Context,
		providerType provider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*provider.Provider, error)
	Gets(ctx context.Context, token string, limit uint64) ([]*provider.Provider, error)
	Delete(ctx context.Context, id uuid.UUID) (*provider.Provider, error)
	Update(
		ctx context.Context,
		id uuid.UUID,
		providerType provider.Type,
		hostname string,
		techPrefix string,
		techPostfix string,
		techHeaders map[string]string,
		name string,
		detail string,
	) (*provider.Provider, error)
}

// NewProviderHandler return ProviderHandler
func NewProviderHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ProviderHandler {
	h := &providerHandler{
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}
