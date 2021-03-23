package webhookhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package webhookhandler -destination ./mock_webhookhandler_webhookhandler.go -source main.go -build_flags=-mod=mod

import (
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

// WebhookHandler is interface for webhook handle
type WebhookHandler interface {
	SendEvent(uri string, method MethodType, dataType DataType, data []byte) (*http.Response, error)
	Test()
}

// webhookHandler structure for service handle
type webhookHandler struct {
	db    dbhandler.DBHandler
	cache cachehandler.CacheHandler
}

var (
	metricsNamespace = "webhook_manager"
)

// DataType defines the send data
type DataType string

// list of DataType
const (
	DataTypeEmpty DataType = ""
	DataTypeJSON  DataType = "application/json"
)

// MethodType defines http method
type MethodType string

// list of Method
const (
	MethodTypePOST   MethodType = "POST"
	MethodTypePUT    MethodType = "PUT"
	MethodTypeGET    MethodType = "GET"
	MethodTypeDELETE MethodType = "DELETE"
)

func init() {
	prometheus.MustRegister()
}

// NewWebhookHandler returns new webhook handler
func NewWebhookHandler(db dbhandler.DBHandler, cache cachehandler.CacheHandler) WebhookHandler {

	h := &webhookHandler{
		db:    db,
		cache: cache,
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
