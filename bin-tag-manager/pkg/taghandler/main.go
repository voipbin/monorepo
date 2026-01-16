package taghandler

//go:generate mockgen -package taghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/dbhandler"
)

// TagHandler interfaces
type TagHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name, detail string) (*tag.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	Get(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	List(ctx context.Context, size uint64, token string, filters map[tag.Field]any) ([]*tag.Tag, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) (*tag.Tag, error)

	EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error
}

type tagHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyhandler notifyhandler.NotifyHandler
}

// NewTagHandler return TagHandler interface
func NewTagHandler(reqHandler requesthandler.RequestHandler, dbHandler dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler) TagHandler {
	return &tagHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyhandler: notifyHandler,
	}
}
