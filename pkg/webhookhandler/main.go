package webhookhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package webhookhandler -destination ./mock_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/messagetargethandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendMessage(uri string, method string, dataType string, data []byte) (*http.Response, error)
	SendWebhook(wh *webhook.Webhook) error
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db dbhandler.DBHandler

	messageTargetHandler messagetargethandler.MessageTargetHandler
}

var (
	metricsNamespace = "webhook_manager"
)

func init() {
	prometheus.MustRegister()
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, messageTargetHandler messagetargethandler.MessageTargetHandler) WebhookHandler {

	h := &webhookHandler{
		db:                   db,
		messageTargetHandler: messageTargetHandler,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// getCurTime return current utc time string
func getCurTimeRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
