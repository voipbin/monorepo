package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/siphandler"
)

var (
	regV1Events  = regexp.MustCompile("/v1/events$")
	regV1SIPInfo = regexp.MustCompile("/v1/sip/info$")
	regV1SIPPcap = regexp.MustCompile("/v1/sip/pcap$")
)

var (
	metricsNamespace = "timeline_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets:   []float64{50, 100, 500, 1000, 3000},
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(promReceivedRequestProcessTime)
}

// ListenHandler interface
type ListenHandler interface {
	Run(queue string) error
}

type listenHandler struct {
	sockHandler  sockhandler.SockHandler
	eventHandler eventhandler.EventHandler
	sipHandler   siphandler.SIPHandler
}

// NewListenHandler creates a new ListenHandler.
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	eventHandler eventhandler.EventHandler,
	sipHandler siphandler.SIPHandler,
) ListenHandler {
	return &listenHandler{
		sockHandler:  sockHandler,
		eventHandler: eventHandler,
		sipHandler:   sipHandler,
	}
}

func simpleResponse(code int) *sock.Response {
	return &sock.Response{StatusCode: code}
}

func (h *listenHandler) Run(queue string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":  "Run",
		"queue": queue,
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "timeline-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	ctx := context.Background()

	var requestType string
	var err error
	var response *sock.Response

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))
	}()

	switch {
	case regV1Events.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/events"
		response, err = h.v1EventsPost(ctx, m)

	case regV1SIPInfo.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/sip/info"
		response, err = h.v1SIPInfoPost(ctx, m)

	case regV1SIPPcap.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/sip/pcap"
		response, err = h.v1SIPPcapPost(ctx, m)

	default:
		log.Errorf("Could not find corresponded request handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	if err != nil {
		log.Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	return response, err
}
