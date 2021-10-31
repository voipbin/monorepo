package confbridgehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package confbridgehandler -destination ./mock_confbridgehandler_confbridgehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

// List of default values
const (
	defaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

// Contexts of confbridge types
const (
	contextConfbridgeIncoming string = "conf-in"
	contextConfbridgeOutgoing string = "conf-out"
)

// ConfbridgeHandler is interface for conference handle
type ConfbridgeHandler interface {
	ARIChannelEnteredBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	ARIChannelLeftBridge(cn *channel.Channel, br *bridge.Bridge) error
	ARIStasisStart(cn *channel.Channel, data map[string]string) error

	Create(ctx context.Context, confID uuid.UUID) (*confbridge.Confbridge, error)
	Join(confbridgeID, callID uuid.UUID) error
	Joined(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	Kick(ctx context.Context, id, callID uuid.UUID) error
	Leaved(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	Terminate(id uuid.UUID) error
}

// confbridgeHandler structure for service handle
type confbridgeHandler struct {
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
	db            dbhandler.DBHandler
	cache         cachehandler.CacheHandler
}

var (
	metricsNamespace = "call_manager"

	promConfbridgeCreateTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "confbridge_create_total",
			Help:      "Total number of created confbridge with type.",
		},
	)

	promConfbridgeCloseTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "confbridge_close_total",
			Help:      "Total number of closed confbridge type.",
		},
	)

	promConfbridgeJoinTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "confbridge_join_total",
			Help:      "Total number of joined calls to the confbridge with type.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promConfbridgeCreateTotal,
		promConfbridgeCloseTotal,
		promConfbridgeJoinTotal,
	)
}

// NewConfbridgeHandler returns new service handler
func NewConfbridgeHandler(req requesthandler.RequestHandler, notify notifyhandler.NotifyHandler, db dbhandler.DBHandler, cache cachehandler.CacheHandler) ConfbridgeHandler {

	h := &confbridgeHandler{
		reqHandler:    req,
		notifyHandler: notify,
		db:            db,
		cache:         cache,
	}

	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

// generateBridgeName generates the bridge name for the confbridge
// all of confbridge creates the bridge must use this function for bridge's name.
func generateBridgeName(referenceType bridge.ReferenceType, confbridgeID uuid.UUID) string {
	res := fmt.Sprintf("reference_type=%s,reference_id=%s", referenceType, confbridgeID.String())

	return res
}
