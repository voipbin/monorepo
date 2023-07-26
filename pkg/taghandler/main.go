package taghandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package taghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cmcustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/tag-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/tag-manager.git/pkg/dbhandler"
)

// TagHandler interfaces
type TagHandler interface {
	Create(ctx context.Context, customerID uuid.UUID, name, detail string) (*tag.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	Get(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	Gets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error)
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
