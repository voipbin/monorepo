package contacthandler

//go:generate mockgen -package contacthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// ContactHandler is interface for service handle
type ContactHandler interface {
	ContactGetsByExtension(ctx context.Context, customerID uuid.UUID, ext string) ([]*astcontact.AstContact, error)
	ContactRefreshByEndpoint(ctx context.Context, customerID uuid.UUID, ext string) error
}

// contactHandler structure for service handle
type contactHandler struct {
	reqHandler requesthandler.RequestHandler
	dbAst      dbhandler.DBHandler
	dbBin      dbhandler.DBHandler
}

func init() {
	prometheus.MustRegister()
}

// NewContactHandler returns new service handler
func NewContactHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler) ContactHandler {

	h := &contactHandler{
		reqHandler: r,
		dbAst:      dbAst,
		dbBin:      dbBin,
	}

	return h
}
