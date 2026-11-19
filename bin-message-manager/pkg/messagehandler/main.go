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
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// list of hook suffix types
const (
	hookTelnyx = "telnyx"
	hookTwilio = "twilio"
)

// MessageHandler defines
type MessageHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)

	Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error)
	SendMMS(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string, medias []uuid.UUID) (*message.Message, error)

	// Hook handles a hook message. signature is the signature header of the provider request (twilio only).
	Hook(ctx context.Context, uri string, m []byte, signature string) error
}

// messageHandler defines
type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	// smsProviders and mmsProviders are the providers by name. They are written once when the handler is created
	// and only read afterwards. The order in which the providers are tried belongs to each message (providerOrder),
	// so these maps are never sorted or changed.
	smsProviders map[message.ProviderName]SMSProvider
	mmsProviders map[message.ProviderName]MMSProvider

	// reqExternal verifies the signature of the twilio callbacks.
	reqExternal requestexternal.RequestExternal

	// twilioAvailable tells whether Twilio sending is configured. Nil means not.
	twilioAvailable func(ctx context.Context) bool

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

		smsProviders: map[message.ProviderName]SMSProvider{
			message.ProviderNameTelnyx:      NewSMSTelnyx(requestExternal),
			message.ProviderNameMessagebird: NewSMSMessagebird(requestExternal),
			message.ProviderNameTwilio:      NewSMSTwilio(requestExternal),
		},
		mmsProviders: map[message.ProviderName]MMSProvider{
			message.ProviderNameTelnyx:      NewMMSTelnyx(requestExternal),
			message.ProviderNameMessagebird: NewMMSMessagebird(requestExternal),
			message.ProviderNameTwilio:      NewMMSTwilio(requestExternal),
		},

		reqExternal:     requestExternal,
		twilioAvailable: requestExternal.TwilioAvailable,

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

	promTwilioSendTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "twilio_number_send_total",
			Help:      "Total number of send message by type.",
		},
		[]string{"type"},
	)

	promTwilioCallbackRejectedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "twilio_callback_rejected_total",
			Help:      "Total number of twilio delivery callbacks that were refused, by reason.",
		},
		[]string{"reason"},
	)

	promMMSTargetFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "mms_target_failed_total",
			Help:      "Total number of mms targets that no provider accepted, by the last provider tried and the reason.",
		},
		[]string{"provider", "reason"},
	)

	promSegmentMismatchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "segment_mismatch_total",
			Help:      "Total number of targets whose segment count reported by an sms provider differs from the platform count.",
		},
		[]string{"provider"},
	)

	promReceiptUnmatchedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "receipt_unmatched_total",
			Help:      "Total number of delivery receipts that matched no message.",
		},
	)

	promFinalWriteFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "dispatch_final_write_failed_total",
			Help:      "Total number of dispatches whose final result could not be written, so the targets stay queued.",
		},
	)

	promEventDroppedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "event_dropped_total",
			Help:      "Total number of message_updated events that were dropped because the message could not be read.",
		},
	)

	promPromotedMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "promoted_messages_total",
			Help:      "Total number of texts promoted to a text only mms, by segment count.",
		},
		[]string{"segments"},
	)

	promPromotedMMSTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "promoted_mms_total",
			Help:      "Total number of mms requests of promoted messages, by provider and outcome.",
		},
		[]string{"provider", "outcome"},
	)

	promPromotedFallbackTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "promoted_fallback_total",
			Help:      "Total number of targets of promoted messages that were sent as sms segments, by reason and segment count.",
		},
		[]string{"reason", "segments"},
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
		promTwilioSendTotal,
		promTwilioCallbackRejectedTotal,
		promMMSTargetFailedTotal,
		promMMSMediaCheckSeconds,
		promInboundMediaTotal,
		promSegmentMismatchTotal,
		promReceiptUnmatchedTotal,
		promFinalWriteFailedTotal,
		promEventDroppedTotal,
		promPromotedMessagesTotal,
		promPromotedMMSTotal,
		promPromotedFallbackTotal,
	)
}
