package listenhandler

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/flowhandler"
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
	db         dbhandler.DBHandler

	flowHandler flowhandler.FlowHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// activeflows
	regV1ActiveFlows       = regexp.MustCompile("/v1/active-flows")
	regV1ActiveFlowsIDNext = regexp.MustCompile("/v1/active-flows/" + regUUID + "/next")

	// flows
	regV1Flows            = regexp.MustCompile("/v1/flows")
	regV1FlowsID          = regexp.MustCompile("/v1/flows/" + regUUID)
	regV1FlowsIDActionsID = regexp.MustCompile("/v1/flows/" + regUUID + "/actions/" + regUUID)
)

var (
	metricsNamespace = "flow_manager"

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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	db dbhandler.DBHandler,
	flowHandler flowhandler.FlowHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:  rabbitSock,
		db:          db,
		flowHandler: flowHandler,
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

	// Set QoS
	if err := h.rabbitSock.QueueQoS(queue, 1, 0); err != nil {
		logrus.Errorf("Could not set the queue's qos. err: %v", err)
		return err
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// process the received request
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPCOpt(queue, "flow-manager", false, false, false, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {

	// v1
	case regV1ActiveFlows.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		requestType = "/active-flows"
		response, err = h.v1ActiveFlowsPost(m)

	case regV1ActiveFlowsIDNext.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/active-flows"
		response, err = h.v1ActiveFlowsIDNextGet(m)

	case regV1FlowsIDActionsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/flows/actions"
		response, err = h.v1FlowsIDActionsIDGet(m)

	case regV1FlowsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/flows"
		response, err = h.v1FlowsIDGet(m)

	case regV1FlowsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPut:
		requestType = "/flows"
		response, err = h.v1FlowsIDPut(m)

	case regV1Flows.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		requestType = "/flows"
		response, err = h.v1FlowsPost(m)

	case regV1FlowsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		requestType = "/flows"
		response, err = h.v1FlowsIDDelete(m)

	case regV1Flows.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		requestType = "/flows"
		response, err = h.v1FlowsGet(m)

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

	// default error handler
	if err != nil {
		logrus.WithFields(
			logrus.Fields{
				"uri":    m.URI,
				"method": m.Method,
				"error":  err,
			}).Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
		requestType = "notfound"
	}

	logrus.WithFields(
		logrus.Fields{
			"response": response,
			"err":      err,
		}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}
