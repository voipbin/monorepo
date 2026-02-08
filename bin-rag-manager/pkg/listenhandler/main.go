package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

// regexp patterns for routing
var (
	regV1RagQuery          = regexp.MustCompile(`^/v1/rags/query$`)
	regV1RagIndex          = regexp.MustCompile(`^/v1/rags/index$`)
	regV1RagIndexIncr      = regexp.MustCompile(`^/v1/rags/index/incremental$`)
	regV1RagIndexStatus    = regexp.MustCompile(`^/v1/rags/index/status$`)
)

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
	var err error

	ctx := context.Background()

	switch {
	// POST /v1/rags/query
	case regV1RagQuery.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagQueryPost(ctx, m)
		requestType = "/v1/rags/query"

	// POST /v1/rags/index
	case regV1RagIndex.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagIndexPost(ctx, m)
		requestType = "/v1/rags/index"

	// POST /v1/rags/index/incremental
	case regV1RagIndexIncr.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1RagIndexIncrementalPost(ctx, m)
		requestType = "/v1/rags/index/incremental"

	// GET /v1/rags/index/status
	case regV1RagIndexStatus.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1RagIndexStatusGet(ctx, m)
		requestType = "/v1/rags/index/status"

	default:
		log.Errorf("Could not find the handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		requestType = "notfound"
	}

	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not process the request correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	return response, err
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
