package contacthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package contacthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
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
