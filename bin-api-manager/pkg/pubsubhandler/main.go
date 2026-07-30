package pubsubhandler

//go:generate mockgen -package pubsubhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

// PubHandler defines the publishing side of the in-process pub/sub.
type PubHandler interface {
	Publish(topic string, message string) error
}

// SubHandler defines a per-connection subscriber of the in-process pub/sub.
// It mirrors the behavioral contract of the previous ZeroMQ SUB socket:
// prefix-based topic matching, deliver-once on overlapping prefixes, and
// drop-on-full slow-subscriber isolation.
type SubHandler interface {
	Terminate()

	Run(ctx context.Context, ws *websocket.Conn) error
	RunWithMutex(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error

	Subscribe(topic string) error
	Unsubscribe(topic string) error
}

// BrokerHandler defines the process-wide in-process pub/sub broker.
// One broker exists per pod; the subscribe path publishes into it and every
// WebSocket connection creates its own SubHandler from it.
type BrokerHandler interface {
	PubHandler

	NewSubHandler() SubHandler
}

// subscriberBufferSize matches the effective capacity of the previous
// ZeroMQ inproc pipe (SNDHWM 1000 + RCVHWM 1000). Messages beyond this are
// dropped for that subscriber only (observable via the drop counter metric).
const subscriberBufferSize = 2000

// message is the unit of delivery: the topic it was published under and the
// payload forwarded to the WebSocket client.
type message struct {
	topic string
	data  string
}

var (
	metricsNamespace = "api_manager"

	promPubsubDroppedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "pubsub_dropped_message_total",
			Help:      "Total number of in-process pub/sub messages dropped because a subscriber buffer was full.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		promPubsubDroppedTotal,
	)
}
