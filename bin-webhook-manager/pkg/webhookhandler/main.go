package webhookhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package webhookhandler -destination ./mock_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/webhook"
	"monorepo/bin-webhook-manager/pkg/accounthandler"
	"monorepo/bin-webhook-manager/pkg/dbhandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendWebhookToCustomer(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error
	SendWebhookToURI(ctx context.Context, customerID uuid.UUID, uri string, method webhook.MethodType, dataType webhook.DataType, data json.RawMessage) error
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler

	accoutHandler accounthandler.AccountHandler
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, notifyHandler notifyhandler.NotifyHandler, messageTargetHandler accounthandler.AccountHandler) WebhookHandler {

	h := &webhookHandler{
		db:            db,
		notifyHandler: notifyHandler,

		accoutHandler: messageTargetHandler,
	}

	return h
}
