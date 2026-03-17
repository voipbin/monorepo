package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-rag-manager/pkg/raghandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ListenHandler interface for rag-manager listen handler
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler
	ragHandler  raghandler.RagHandler
}

var promReceivedRequestProcessTime = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "bin_rag_manager",
		Name:      "receive_request_process_time",
		Help:      "Process time of received requests",
		Buckets:   []float64{50, 100, 500, 1000, 3000, 5000, 10000},
	},
	[]string{"type", "method"},
)

func init() {
	prometheus.MustRegister(promReceivedRequestProcessTime)
}

// NewListenHandler creates a new ListenHandler
func NewListenHandler(sockHandler sockhandler.SockHandler, ragHandler raghandler.RagHandler) ListenHandler {
	return &listenHandler{
		sockHandler: sockHandler,
		ragHandler:  ragHandler,
	}
}

// Run starts listening for RPC requests
func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithField("func", "Run")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not create queue: %w", err)
	}

	go func() {
		if err := h.sockHandler.ConsumeRPC(context.Background(), queue, "rag-manager", false, false, false, 10, h.processRequest); err != nil {
			log.Errorf("Could not consume RPC. err: %v", err)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "processRequest",
		"uri":    m.URI,
		"method": m.Method,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	var requestType string
	var response *sock.Response

	switch {
	default:
		log.Errorf("Could not find the handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		requestType = "notfound"
	}

	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, nil
}

func simpleResponse(statusCode int) *sock.Response {
	return &sock.Response{
		StatusCode: statusCode,
	}
}

func jsonResponse(statusCode int, data any) *sock.Response {
	m, err := json.Marshal(data)
	if err != nil {
		return simpleResponse(500)
	}
	return &sock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       m,
	}
}
