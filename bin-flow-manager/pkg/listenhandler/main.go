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

	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
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
	utilHandler utilhandler.UtilHandler
	rabbitSock  rabbitmqhandler.Rabbit

	flowHandler       flowhandler.FlowHandler
	activeflowHandler activeflowhandler.ActiveflowHandler
	variableHandler   variablehandler.VariableHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = ".*"

	// activeflows
	regV1ActiveflowsGet               = regexp.MustCompile(`/v1/activeflows\?`)
	regV1Activeflows                  = regexp.MustCompile("/v1/activeflows$")
	regV1ActiveflowsID                = regexp.MustCompile("/v1/activeflows/" + regUUID + "$")
	regV1ActiveflowsIDExecute         = regexp.MustCompile("/v1/activeflows/" + regUUID + "/execute$")
	regV1ActiveflowsIDNext            = regexp.MustCompile("/v1/activeflows/" + regUUID + "/next$")
	regV1ActiveflowsIDForwardActionID = regexp.MustCompile("/v1/activeflows/" + regUUID + "/forward_action_id$")
	regV1ActiveflowsIDStop            = regexp.MustCompile("/v1/activeflows/" + regUUID + "/stop$")
	regV1ActiveflowsIDPushActions     = regexp.MustCompile("/v1/activeflows/" + regUUID + "/push_actions$")

	// flows
	regV1FlowsGet         = regexp.MustCompile(`/v1/flows\?`)
	regV1Flows            = regexp.MustCompile("/v1/flows$")
	regV1FlowsID          = regexp.MustCompile("/v1/flows/" + regUUID + "$")
	regV1FlowsIDActions   = regexp.MustCompile("/v1/flows/" + regUUID + "/actions$")
	regV1FlowsIDActionsID = regexp.MustCompile("/v1/flows/" + regUUID + "/actions/" + regUUID + "$")

	// variables
	regV1VariablesID             = regexp.MustCompile("/v1/variables/" + regUUID + "$")
	regV1VariablesIDVariables    = regexp.MustCompile("/v1/variables/" + regUUID + "/variables$")
	regV1VariablesIDVariablesKey = regexp.MustCompile("/v1/variables/" + regUUID + "/variables/" + regAny + "$")
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
	flowHandler flowhandler.FlowHandler,
	activeflowHandler activeflowhandler.ActiveflowHandler,
	variableHandler variablehandler.VariableHandler,
) ListenHandler {
	h := &listenHandler{
		utilHandler:       utilhandler.NewUtilHandler(),
		rabbitSock:        rabbitSock,
		flowHandler:       flowHandler,
		activeflowHandler: activeflowHandler,
		variableHandler:   variableHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Run",
		"queue":          queue,
		"exchange_delay": exchangeDelay,
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.rabbitSock.QueueDeclare(queue, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// Set QoS
	if err := h.rabbitSock.QueueQoS(queue, 1, 0); err != nil {
		log.Errorf("Could not set the queue's qos. err: %v", err)
		return err
	}

	// create a exchange for delayed message
	if err := h.rabbitSock.ExchangeDeclareForDelay(exchangeDelay, true, false, false, false); err != nil {
		return fmt.Errorf("could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("could not bind the queue and exchange. err: %v", err)
	}

	// process the received request
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPCOpt(queue, "flow-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				log.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*rabbitmqhandler.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	ctx := context.Background()

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))
	}()

	log.Debugf("Received request. uri: %s, method: %s", m.URI, m.Method)
	switch {

	// v1
	// activeflows
	case regV1ActiveflowsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/activeflows"
		response, err = h.v1ActiveflowsGet(ctx, m)

	// activeflows
	case regV1Activeflows.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/activeflows"
		response, err = h.v1ActiveflowsPost(ctx, m)

	// activeflows/<activeflow-id>
	case regV1ActiveflowsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/activeflows/<activeflow-id>"
		response, err = h.v1ActiveflowsIDDelete(ctx, m)

	// activeflows/<activeflow-id>
	case regV1ActiveflowsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/activeflows/<activeflow-id>"
		response, err = h.v1ActiveflowsIDGet(ctx, m)

	// activeflows/<activeflow-id>/next
	case regV1ActiveflowsIDNext.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/activeflows/<activeflow-id>/next"
		response, err = h.v1ActiveflowsIDNextGet(ctx, m)

	// activeflows/<activeflow-id>/forward_action_id
	case regV1ActiveflowsIDForwardActionID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/activeflows/<activeflow-id>/forward_action_id"
		response, err = h.v1ActiveflowsIDForwardActionIDPut(ctx, m)

	// activeflows/<activeflow-id>/execute
	case regV1ActiveflowsIDExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/activeflows/<activeflow-id>/execute"
		response, err = h.v1ActiveflowsIDExecutePost(ctx, m)

	// activeflows/<activeflow-id>/stop
	case regV1ActiveflowsIDStop.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/activeflows/<activeflow-id>/stop"
		response, err = h.v1ActiveflowsIDStopPost(ctx, m)

	// activeflows/<activeflow-id>/push_actions
	case regV1ActiveflowsIDPushActions.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/activeflows/<activeflow-id>/push_actions"
		response, err = h.v1ActiveflowsIDPushActionsPost(ctx, m)

	// flows
	case regV1Flows.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/flows"
		response, err = h.v1FlowsPost(ctx, m)

	case regV1FlowsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/flows"
		response, err = h.v1FlowsGet(ctx, m)

	// flows/<flow-id>
	case regV1FlowsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/flows/<flow-id>"
		response, err = h.v1FlowsIDGet(ctx, m)

	case regV1FlowsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/flows/<flow-id>"
		response, err = h.v1FlowsIDPut(ctx, m)

	case regV1FlowsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/flows/<flow-id>"
		response, err = h.v1FlowsIDDelete(ctx, m)

	// flows/<flow-id>/actions
	case regV1FlowsIDActions.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/flows/<flow-id>/actions"
		response, err = h.v1FlowsIDActionsPut(ctx, m)

	case regV1FlowsIDActionsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/flows/<flow-id>/actions/<action-id>"
		response, err = h.v1FlowsIDActionsIDGet(ctx, m)

	// variables/<variable-id>
	case regV1VariablesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/variables/<variable-id>"
		response, err = h.v1VariablesIDGet(ctx, m)

	// variables/<variable-id>/variables
	case regV1VariablesIDVariables.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/variables/<variable-id>/variables"
		response, err = h.v1VariablesIDVariablesPost(ctx, m)

	// variables/<variable-id>/variables/key
	case regV1VariablesIDVariablesKey.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/variables/<variable-id>/variables/key"
		response, err = h.v1VariablesIDVariablesKeyDelete(ctx, m)

	default:
		log.Errorf("Could not find corresponded request handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}

	// default error handler
	if err != nil {
		log.Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	log.WithField("response", response).Debugf("Response the request. uri: %s, method: %s", m.URI, m.Method)

	return response, err
}
