package arieventlistenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package arieventlistenhandler -destination ./mock_arieventlistenhandler.go -source main.go -build_flags=-mod=mod

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/arieventhandler"
)

// ARIEventListenHandler intreface for ARI event listen handler
type ARIEventListenHandler interface {
	Run(queue, receiver string) error
}

type ariEventListenHandler struct {
	rabbitSock rabbitmqhandler.Rabbit

	ariEventHandler arieventhandler.ARIEventHandler
}

var (
	metricsNamespace = "call_manager"

	promARIEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_total",
			Help:      "Total number of received ARI event types.",
		},
		[]string{"type", "asterisk_id"},
	)

	promARIProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "ari_event_listen_process_time",
			Help:      "Process time of received ARI events",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"asterisk_id", "type"},
	)
)

func init() {
	prometheus.MustRegister(
		promARIEventTotal,
		promARIProcessTime,
	)

}

// NewARIEventListenHandler create EventHandler
func NewARIEventListenHandler(
	sock rabbitmqhandler.Rabbit,
	ariEventHandler arieventhandler.ARIEventHandler,
) ARIEventListenHandler {
	h := &ariEventListenHandler{
		rabbitSock:      sock,
		ariEventHandler: ariEventHandler,
	}

	return h
}

// Run starts to receive ARI event and process it.
func (h *ariEventListenHandler) Run(queue, receiver string) error {
	// create queue for ari event receive
	log := logrus.WithFields(logrus.Fields{
		"queue": queue,
	})

	log.Infof("Creating rabbitmq queue for ARI event receiving.")

	err := h.rabbitSock.QueueDeclare(queue, true, false, false, false)
	if err != nil {
		return err
	}

	// Set QoS
	if err := h.rabbitSock.QueueQoS(queue, 1, 0); err != nil {
		log.Errorf("Could not set the queue's qos. err: %v", err)
		return err
	}

	// receive ARI event
	go func() {
		for {
			if err := h.rabbitSock.ConsumeMessageOpt(queue, receiver, false, false, false, h.processEventRun); err != nil {
				log.Errorf("Could not consume the message. err: %v", err)
			}
		}
	}()
	return nil
}

func (h *ariEventListenHandler) processEventRun(m *rabbitmqhandler.Event) error {
	go func() {
		if err := h.processEvent(m); err != nil {
			logrus.Errorf("Could not consume the ARI event message correctly. err: %v", err)
		}
	}()

	return nil
}
