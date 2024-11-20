package messagehandlermessagebird

//go:generate mockgen -package messagehandlermessagebird -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

// MessageHandlerMessagebird is interface for service handle
type MessageHandlerMessagebird interface {
	SendMessage(messageID uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, targets []target.Target, text string) ([]target.Target, error)
}

// messageHandlerMessagebird structure for service handle
type messageHandlerMessagebird struct {
	reqHandler requesthandler.RequestHandler
	db         dbhandler.DBHandler

	requestExternal requestexternal.RequestExternal
}

var (
	metricsNamespace = "message_manager"

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
		promMessagebirdSendTotal,
	)
}

// NewMessageHandlerMessagebird returns new service handler
func NewMessageHandlerMessagebird(r requesthandler.RequestHandler, db dbhandler.DBHandler, reqExternal requestexternal.RequestExternal) MessageHandlerMessagebird {
	h := &messageHandlerMessagebird{
		reqHandler:      r,
		db:              db,
		requestExternal: reqExternal,
	}

	return h
}
