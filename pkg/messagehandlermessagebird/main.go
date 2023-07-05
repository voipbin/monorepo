package messagehandlermessagebird

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package messagehandlermessagebird -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/requestexternal"
)

// MessageHandlerMessagebird is interface for service handle
type MessageHandlerMessagebird interface {
	// SendMessage(messageID uuid.UUID, customerID uuid.UUID, source *commonaddress.Address, destinations []commonaddress.Address, text string) (*message.Message, error)
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

	promMessagebirdCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "messagebird_number_create_total",
			Help:      "Total number of created number type by messagebird.",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(
		promMessagebirdCreateTotal,
	)
}

// NewMessageHandlerMessagebird returns new service handler
func NewMessageHandlerMessagebird(r requesthandler.RequestHandler, db dbhandler.DBHandler) MessageHandlerMessagebird {

	reqExternal := requestexternal.NewRequestExternal()

	h := &messageHandlerMessagebird{
		reqHandler:      r,
		db:              db,
		requestExternal: reqExternal,
	}

	return h
}
