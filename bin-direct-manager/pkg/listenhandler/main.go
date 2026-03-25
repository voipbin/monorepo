package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-direct-manager/pkg/directhandler"
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
	sockHandler sockhandler.SockHandler

	utilHandler    utilhandler.UtilHandler
	directHandler  directhandler.DirectHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok

	// v1
	// directs
	regV1DirectsPost         = regexp.MustCompile(`/v1/directs$`)
	regV1DirectsGet          = regexp.MustCompile(`/v1/directs\?`)
	regV1DirectsByHashGet    = regexp.MustCompile(`/v1/directs/by-hash/`)
	regV1DirectsIDRegenerate = regexp.MustCompile(`/v1/directs/` + regUUID + `/regenerate$`)
	regV1DirectsID           = regexp.MustCompile(`/v1/directs/` + regUUID + `$`)
)

var (
	metricsNamespace = "direct_manager"

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
func NewListenHandler(sockHandler sockhandler.SockHandler, directHandler directhandler.DirectHandler) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		utilHandler:    utilhandler.NewUtilHandler(),
		directHandler:  directHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.WithFields(logrus.Fields{
		"queue":          queue,
		"exchange_delay": exchangeDelay,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "direct-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// directs
	////////////
	// GET /directs?...
	case regV1DirectsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DirectsGet(ctx, m)
		requestType = "/v1/directs"

	// POST /directs
	case regV1DirectsPost.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1DirectsPost(ctx, m)
		requestType = "/v1/directs"

	// GET /directs/by-hash/{hash} (must be before ID get)
	case regV1DirectsByHashGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DirectsByHashGet(ctx, m)
		requestType = "/v1/directs/by-hash"

	// POST /directs/{id}/regenerate (must be before ID get)
	case regV1DirectsIDRegenerate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1DirectsIDRegenerate(ctx, m)
		requestType = "/v1/directs/regenerate"

	// DELETE /directs/{direct-id}
	case regV1DirectsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1DirectsIDDelete(ctx, m)
		requestType = "/v1/directs"

	// GET /directs/{direct-id}
	case regV1DirectsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1DirectsIDGet(ctx, m)
		requestType = "/v1/directs"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(400)
		err = nil
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)
	}

	return response, err
}
