package arihandler

import (
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// ARIHandler arihandler package interface
type ARIHandler interface {
	Connect(addr, rabbitQueueARIEvent string)
	Run()
}

type ariHandler struct {
	rabbitQueueARIEvent string

	rabbitSock rabbitmq.Rabbit

	reqHandler RequestHandler
	evtHandler EventHandler
}

var (
	metricsNamespace = "call_manager"

	promARIEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_total",
			Help:      "Total number of received ARI event types.",
		},
		[]string{"type", "asterisk_id"},
	)
)

func init() {
	prometheus.MustRegister(promARIEventTotal)
}

// NewARIHandler creates ARIHandler interface
func NewARIHandler() ARIHandler {
	handler := &ariHandler{}

	handler.reqHandler = NewRequestHandler()
	handler.evtHandler = NewEventHandler()

	return handler
}

// Connect connects to rabbitmq
func (h *ariHandler) Connect(addr, rabbitQueueARIEvent string) {
	// create queue for ari event receive
	log.WithFields(log.Fields{
		"addr": addr,
	}).Infof("Connecting to the rabbitmq.")

	h.rabbitSock = rabbitmq.NewRabbit(addr)
	h.rabbitQueueARIEvent = rabbitQueueARIEvent

	// connect
	h.rabbitSock.Connect()

	// set sock for request handler
	h.evtHandler.SetSock(h.rabbitSock)
	h.reqHandler.SetSock(h.rabbitSock)
}

// Run runs the arihandler
func (h *ariHandler) Run() {
	h.evtHandler.HandleARIEvent(h.rabbitQueueARIEvent, "call-manager")
}
