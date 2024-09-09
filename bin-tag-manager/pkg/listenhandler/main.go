package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-tag-manager/pkg/taghandler"
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
	rabbitSock rabbitmqhandler.Rabbit

	utilHandler utilhandler.UtilHandler
	tagHandler  taghandler.TagHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok

	// v1
	// tags
	regV1Tags    = regexp.MustCompile("/v1/tags$")
	regV1TagsGet = regexp.MustCompile(`/v1/tags\?(.*)$`)
	regV1TagsID  = regexp.MustCompile("/v1/tags/" + regUUID + "$")
)

var (
	metricsNamespace = "agent_manager"

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
func NewListenHandler(rabbitSock rabbitmqhandler.Rabbit, tagHandler taghandler.TagHandler) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,

		utilHandler: utilhandler.NewUtilHandler(),
		tagHandler:  tagHandler,
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

	if err := h.rabbitSock.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPC(queue, "tag-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				log.Errorf("Could not consume the request message correctly. err: %v", err)
			}
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
	// tags
	////////////
	// GET /tags
	case regV1TagsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TagsGet(ctx, m)
		requestType = "/v1/tags"

	// POST /tags
	case regV1Tags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1TagsPost(ctx, m)
		requestType = "/v1/tags"

	// DELETE /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1TagsIDDelete(ctx, m)
		requestType = "/v1/tags"

	// GET /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TagsIDGet(ctx, m)
		requestType = "/v1/tags"

	// PUT /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1TagsIDPut(ctx, m)
		requestType = "/v1/tags"

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
