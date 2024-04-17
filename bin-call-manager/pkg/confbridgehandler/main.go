package confbridgehandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package confbridgehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/externalmedia"
	commonoutline "gitlab.com/voipbin/bin-manager/common-handler.git/models/outline"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cucustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/bridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/cachehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/externalmediahandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/recordinghandler"
)

// ConfbridgeHandler is interface for conference handle
type ConfbridgeHandler interface {
	ARIChannelEnteredBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	ARIChannelLeftBridge(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	ARIStasisStart(ctx context.Context, cn *channel.Channel) error
	ARIBridgeDestroyed(ctx context.Context, br *bridge.Bridge) error

	Create(ctx context.Context, customerID uuid.UUID, confbridgeType confbridge.Type) (*confbridge.Confbridge, error)
	Delete(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	UpdateExternalMediaID(ctx context.Context, id uuid.UUID, externalMediaID uuid.UUID) (*confbridge.Confbridge, error)
	Get(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)
	Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*confbridge.Confbridge, error)
	Join(ctx context.Context, confbridgeID, callID uuid.UUID) error
	Joined(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	Kick(ctx context.Context, id, callID uuid.UUID) error
	Leaved(ctx context.Context, cn *channel.Channel, br *bridge.Bridge) error
	Terminating(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)

	Ring(ctx context.Context, id uuid.UUID) error
	Answer(ctx context.Context, id uuid.UUID) error

	RecordingStart(ctx context.Context, id uuid.UUID, format recording.Format, endOfSilence int, endOfKey string, duration int) (*confbridge.Confbridge, error)
	RecordingStop(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)

	ExternalMediaStart(ctx context.Context, id uuid.UUID, externalHost string, encapsulation externalmedia.Encapsulation, transport externalmedia.Transport, connectionType string, format string, direction string) (*confbridge.Confbridge, error)
	ExternalMediaStop(ctx context.Context, id uuid.UUID) (*confbridge.Confbridge, error)

	FlagAdd(ctx context.Context, id uuid.UUID, flag confbridge.Flag) (*confbridge.Confbridge, error)
	FlagRemove(ctx context.Context, id uuid.UUID, flag confbridge.Flag) (*confbridge.Confbridge, error)

	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

// confbridgeHandler structure for service handle
type confbridgeHandler struct {
	utilHandler          utilhandler.UtilHandler
	reqHandler           requesthandler.RequestHandler
	notifyHandler        notifyhandler.NotifyHandler
	db                   dbhandler.DBHandler
	cache                cachehandler.CacheHandler
	channelHandler       channelhandler.ChannelHandler
	bridgeHandler        bridgehandler.BridgeHandler
	recordingHandler     recordinghandler.RecordingHandler
	externalMediaHandler externalmediahandler.ExternalMediaHandler
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(common.Servicename)

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
func NewConfbridgeHandler(
	req requesthandler.RequestHandler,
	notify notifyhandler.NotifyHandler,
	db dbhandler.DBHandler,
	cache cachehandler.CacheHandler,
	channelHandler channelhandler.ChannelHandler,
	bridgeHandler bridgehandler.BridgeHandler,
	recordingHandler recordinghandler.RecordingHandler,
	externalMediaHandler externalmediahandler.ExternalMediaHandler,
) ConfbridgeHandler {

	h := &confbridgeHandler{
		utilHandler:          utilhandler.NewUtilHandler(),
		reqHandler:           req,
		notifyHandler:        notify,
		db:                   db,
		cache:                cache,
		channelHandler:       channelHandler,
		bridgeHandler:        bridgeHandler,
		recordingHandler:     recordingHandler,
		externalMediaHandler: externalMediaHandler,
	}

	return h
}

// generateBridgeName generates the bridge name for the confbridge
// all of confbridge creates the bridge must use this function for bridge's name.
func generateBridgeName(referenceType bridge.ReferenceType, confbridgeID uuid.UUID) string {
	res := fmt.Sprintf("reference_type=%s,reference_id=%s", referenceType, confbridgeID.String())

	return res
}
