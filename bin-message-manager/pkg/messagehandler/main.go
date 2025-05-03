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
	Gets(ctx context.Context, customerID uuid.UUID, pageSize uint64, pageToken string) ([]*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)

	Send(ctx context.Context, id uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error)

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
)

func init() {
	prometheus.MustRegister(
		promTelnyxSendTotal,
		promMessagebirdSendTotal,
	)
}
