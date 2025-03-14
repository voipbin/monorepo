package emailhandler

//go:generate mockgen -package emailhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
)

type emailHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	engineSendgrid EngineSendgrid
}

type EmailHandler interface {
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		destinations []commonaddress.Address,
		subject string,
		content string,
		attachments []email.Attachment,
	) (*email.Email, error)
	Get(ctx context.Context, id uuid.UUID) (*email.Email, error)
	Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*email.Email, error)
	Delete(ctx context.Context, id uuid.UUID) (*email.Email, error)

	Hook(ctx context.Context, uri string, data []byte) error
}

var (
	defaultSource = &commonaddress.Address{
		Type:       commonaddress.TypeEmail,
		Target:     "service@voipbin.net",
		TargetName: "voipbin service",
	}
)

const (
	hookSendgrid = "sendgrid"
)

func NewEmailHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	sendgridAPIKey string,
) EmailHandler {

	engineSendgrid := NewEngineSendgrid(reqHandler, sendgridAPIKey)

	h := &emailHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		engineSendgrid: engineSendgrid,
	}
	return h
}
