package listenhandler

import (
	"fmt"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/flowhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager/pkg/rabbitmq"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	rabbitSock rabbitmq.Rabbit
	db         dbhandler.DBHandler
	// cache      cachehandler.CacheHandler

	flowHandler flowhandler.FlowHandler
	// reqHandler        requesthandler.RequestHandler
	// callHandler       callhandler.CallHandler
	// conferenceHandler conferencehandler.ConferenceHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// flows
	regV1Flows            = regexp.MustCompile("/v1/flows")
	regV1FlowsID          = regexp.MustCompile("/v1/flows/" + regUUID)
	regV1FlowsIDActionsID = regexp.MustCompile("/v1/flows/" + regUUID + "/actions/" + regUUID)
)

var (
	metricsNamespace = "api_manager"

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
func simpleResponse(code int) *rabbitmq.Response {
	return &rabbitmq.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmq.Rabbit,
	db dbhandler.DBHandler,
	flowHandler flowhandler.FlowHandler,
	// cache cachehandler.CacheHandler,
	// reqHandler requesthandler.RequestHandler,
	// callHandler callhandler.CallHandler,
	// conferenceHandler conferencehandler.ConferenceHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:  rabbitSock,
		db:          db,
		flowHandler: flowHandler,
		// cache:             cache,
		// reqHandler:        reqHandler,
		// callHandler:       callHandler,
		// conferenceHandler: conferenceHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.rabbitSock.QueueDeclare(queue, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// receive ARI event
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPC(queue, "call-manager", h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the ARI message correctly. Will try again after 1 second. err: %v", err)
				time.Sleep(time.Second * 1)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmq.Request) (*rabbitmq.Response, error) {

	var requestType string
	var err error
	var response *rabbitmq.Response

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debug("Received request.")

	start := time.Now()
	switch {

	// v1
	case regV1FlowsIDActionsID.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodGet:
		requestType = "/flows/actions"
		return h.v1FlowsIDActionsIDGet(m)

	case regV1FlowsID.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodGet:
		requestType = "/flows"
		return h.v1FlowsIDGet(m)

	case regV1Flows.MatchString(m.URI) == true && m.Method == rabbitmq.RequestMethodPost:
		requestType = "/flows"
		return h.v1FlowsPost(m)

	default:
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
			}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	return response, err
}
