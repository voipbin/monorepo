package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-timeline-manager/pkg/dbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
	"monorepo/bin-timeline-manager/pkg/siphandler"
)

var (
	regUUID = "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"

	regV1Events           = regexp.MustCompile("/v1/events$")
	regV1AggregatedEvents = regexp.MustCompile("/v1/aggregated-events$")
	regV1Correlations     = regexp.MustCompile("/v1/correlations/" + regUUID + "$")
	regV1SIPAnalysis      = regexp.MustCompile("/v1/sip/analysis$")
	regV1SIPPcap          = regexp.MustCompile("/v1/sip/pcap$")
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

// errorResponse maps a business-handler error to the appropriate sock.Response.
// Resolution order: typed *cerrors.VoipbinError → ToResponse; legacy
// dbhandler.ErrNotFound → 404; else → 500.
func errorResponse(err error) *sock.Response {
	if err == nil {
		logrus.WithField("func", "errorResponse").Warn("errorResponse called with nil error — likely a caller bug; returning 500")
		return simpleResponse(http.StatusInternalServerError)
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		resp, e := cerrors.ToResponse(ve)
		if e == nil {
			return resp
		}
		logrus.WithField("func", "errorResponse").Errorf("cerrors.ToResponse failed for typed VoipbinError: %v", e)
		return simpleResponse(http.StatusInternalServerError)
	}

	if stderrors.Is(err, dbhandler.ErrNotFound) {
		return simpleResponse(http.StatusNotFound)
	}

	return simpleResponse(http.StatusInternalServerError)
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

	case regV1AggregatedEvents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/aggregated-events"
		response, err = h.v1AggregatedEventsPost(ctx, m)

	case regV1Correlations.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/correlations"
		response, err = h.v1CorrelationsGet(ctx, m)

	case regV1SIPAnalysis.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/sip/analysis"
		response, err = h.v1SIPAnalysisPost(ctx, m)

	case regV1SIPPcap.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/sip/pcap"
		response, err = h.v1SIPPcapPost(ctx, m)

	default:
		log.Errorf("Could not find corresponded request handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.Errorf("Could not process the request correctly. data: %s", m.Data)
		var ve *cerrors.VoipbinError
		switch {
		case stderrors.As(err, &ve):
			response = errorResponse(err)
		case stderrors.Is(err, dbhandler.ErrNotFound):
			response = errorResponse(err)
		default:
			response = simpleResponse(400)
		}
		err = nil
	}

	return response, err
}
