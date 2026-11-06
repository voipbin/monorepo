package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/messagehandlermessagebird"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// list of hook suffix types
const (
	hookTelnyx = "telnyx"
)

// MessageHandler defines
type MessageHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)

	Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error)
	SendMMS(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string, medias []uuid.UUID) (*message.Message, error)

	Hook(ctx context.Context, uri string, m []byte) error
}

// messageHandler defines
type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	messageHandlerMessagebird messagehandlermessagebird.MessageHandlerMessagebird
	messageHandlerTelnyx      MessageHandlerTelnyx

	mmsTelnyx      MMSProvider
	mmsMessagebird MMSProvider

	// mediaFetcher downloads the media of a received message and ingestSlots limits the ingests that run at the same time.
	mediaFetcher MediaFetcher
	ingestSlots  chan struct{}
}

// NewMessageHandler returns a new MessageHandler
func NewMessageHandler(r requesthandler.RequestHandler, n notifyhandler.NotifyHandler, db dbhandler.DBHandler, requestExternal requestexternal.RequestExternal) MessageHandler {

	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    r,
		notifyHandler: n,

		messageHandlerMessagebird: NewMessageHandlerMessagebird(requestExternal),
		messageHandlerTelnyx:      NewMessageHandlerTelnyx(requestExternal),

		mmsTelnyx:      NewMMSTelnyx(requestExternal),
		mmsMessagebird: NewMMSMessagebird(requestExternal),

		mediaFetcher: NewMediaFetcher(),
		ingestSlots:  make(chan struct{}, inboundMediaMaxIngest),
	}
}

var (
	metricsNamespace = "message_manager"

	promTelnyxSendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "telnyx_number_send_total",
			Help:      "Total number of send message by type.",
		},
		[]string{"type"},
	)

	promMessagebirdSendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "messagebird_number_send_total",
			Help:      "Total number of send message by type.",
		},
		[]string{"type"},
	)

	promMMSTargetFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "mms_target_failed_total",
			Help:      "Total number of mms targets that no provider accepted, by the last provider tried and the reason.",
		},
		[]string{"provider", "reason"},
	)

	promInboundMediaTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "inbound_media_total",
			Help:      "Total number of media of received messages by result.",
		},
		[]string{"result"},
	)

	promMMSMediaCheckSeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "mms_media_check_seconds",
			Help:      "Duration of the media validation and the signed url refresh of one mms request.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
	)
)

func init() {
	prometheus.MustRegister(
		promTelnyxSendTotal,
		promMessagebirdSendTotal,
		promMMSTargetFailedTotal,
		promMMSMediaCheckSeconds,
		promInboundMediaTotal,
	)
}
