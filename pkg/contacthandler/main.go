package contacthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package contacthandler -destination ./mock_contacthandler_contacthandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/dbhandler"
)

// ContactHandler is interface for service handle
type ContactHandler interface {
	ContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error)
	ContactRefreshByEndpoint(ctx context.Context, endpoint string) error
}

// contactHandler structure for service handle
type contactHandler struct {
	reqHandler requesthandler.RequestHandler
	dbAst      dbhandler.DBHandler
	dbBin      dbhandler.DBHandler
	cache      cachehandler.CacheHandler
}

var (
	metricsNamespace = "registrar_manager"
)

func init() {
	prometheus.MustRegister()
}

// NewContactHandler returns new service handler
func NewContactHandler(r requesthandler.RequestHandler, dbAst dbhandler.DBHandler, dbBin dbhandler.DBHandler, cache cachehandler.CacheHandler) ContactHandler {

	h := &contactHandler{
		reqHandler: r,
		dbAst:      dbAst,
		dbBin:      dbBin,
		cache:      cache,
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

func getStringPointer(v string) *string {
	return &v
}

func getIntegerPointer(v int) *int {
	return &v
}
