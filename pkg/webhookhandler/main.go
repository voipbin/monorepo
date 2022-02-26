package webhookhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package webhookhandler -destination ./mock_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/messagetargethandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendWebhook(ctx context.Context, customerID uuid.UUID, dataType webhook.DataType, data json.RawMessage) error
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db dbhandler.DBHandler

	messageTargetHandler messagetargethandler.MessagetargetHandler
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, messageTargetHandler messagetargethandler.MessagetargetHandler) WebhookHandler {

	h := &webhookHandler{
		db:                   db,
		messageTargetHandler: messageTargetHandler,
	}

	return h
}
