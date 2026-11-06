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
	"time"

	"github.com/gofrs/uuid"
)

type emailHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	engineSendgrid EngineSendgrid
	engineMailgun  EngineMailgun

	// mailgunWebhookSigningKey verifies the Mailgun webhook signature.
	// An empty key rejects every Mailgun webhook.
	mailgunWebhookSigningKey string

	// timeNow is the clock of the webhook timestamp check. nil means time.Now.
	timeNow func() time.Time
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
	List(ctx context.Context, token string, size uint64, filters map[email.Field]any) ([]*email.Email, error)
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
	hookMailgun  = "mailgun"
)

func NewEmailHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	sendgridAPIKey string,
	mailgunAPIKey string,
	mailgunWebhookSigningKey string,
) EmailHandler {

	engineSendgrid := NewEngineSendgrid(reqHandler, sendgridAPIKey)
	engineMailgun := NewEngineMailgun(reqHandler, mailgunAPIKey)

	h := &emailHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		engineSendgrid: engineSendgrid,
		engineMailgun:  engineMailgun,

		mailgunWebhookSigningKey: mailgunWebhookSigningKey,
	}
	return h
}
