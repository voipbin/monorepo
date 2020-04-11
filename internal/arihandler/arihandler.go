package arihandler

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// ARIHandler arihandler package interface
type ARIHandler interface {
	ReceiveEventQueue(addr, queue, receiver string)
}

// ARIEvent is the structure for ARI event parse.
type ARIEvent struct {
	eventType   string
	application string
	asteriskID  string
	timestamp   time.Time
	event       interface{}
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
