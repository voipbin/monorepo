package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tts-manager/pkg/streaminghandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler      sockhandler.SockHandler
	ttsHandler       ttshandler.TTSHandler
	streamingHandler streaminghandler.StreamingHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// speeches
	regV1Speeches = regexp.MustCompile("/v1/speeches")

	// streamings
	resV1Streamings      = regexp.MustCompile("/v1/streamings$")
	resV1StreamingsID    = regexp.MustCompile("/v1/streamings/" + regUUID + "$")
	resV1StreamingsIDSay = regexp.MustCompile("/v1/streamings/" + regUUID + "/say$")
)

var (
	metricsNamespace = "tts_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promReceivedRequestProcessTime,
	)
}

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	ttsHandler ttshandler.TTSHandler,
	streamingHandler streaminghandler.StreamingHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:      sockHandler,
		ttsHandler:       ttsHandler,
		streamingHandler: streamingHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errRPC := h.sockHandler.ConsumeRPC(context.Background(), queue, "tts-manager", false, false, false, 10, h.processRequest); errRPC != nil {
			logrus.Errorf("Could not consume the message correctly. err: %v", errRPC)
		}
	}()

	return nil
}

// processRequest
func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	var requestType string
	var err error
	var response *sock.Response

	log.Debug("Received request.")

	ctx := context.Background()
	start := time.Now()
	switch {

	////// v1
	// /speeches
	case regV1Speeches.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/speeches"
		response, err = h.v1SpeechesPost(ctx, m)

	// /streamings
	case resV1Streamings.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/streamings"
		response, err = h.v1StreamingsPost(ctx, m)

	// /streamings/<id>
	case resV1StreamingsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/streamings/<streaming-id>"
		response, err = h.v1StreamingsIDDelete(ctx, m)

	// /streamings/<id>/say
	case resV1StreamingsIDSay.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/streamings/<streaming-id>/say"
		response, err = h.v1StreamingsIDSayPost(ctx, m)

	default:
		log.Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, err
}
