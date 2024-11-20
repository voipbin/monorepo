package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-agent-manager/pkg/agenthandler"
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
	sockHandler sockhandler.SockHandler

	agentHandler agenthandler.AgentHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok
	regAny  = "(.*)"

	// v1
	// agents
	regV1Agents                     = regexp.MustCompile("/v1/agents$")
	regV1AgentsGet                  = regexp.MustCompile(`/v1/agents\?(.*)$`)
	regV1AgentsUsernameLogin        = regexp.MustCompile("/v1/agents/" + regAny + "/login$")
	regV1AgentsID                   = regexp.MustCompile("/v1/agents/" + regUUID + "$")
	regV1AgentsIDAddresses          = regexp.MustCompile("/v1/agents/" + regUUID + "/addresses$")
	regV1AgentsIDTagIDs             = regexp.MustCompile("/v1/agents/" + regUUID + "/tag_ids$")
	regV1AgentsIDStatus             = regexp.MustCompile("/v1/agents/" + regUUID + "/status$")
	regV1AgentsIDPassword           = regexp.MustCompile("/v1/agents/" + regUUID + "/password$")
	regV1AgentsIDPermission         = regexp.MustCompile("/v1/agents/" + regUUID + "/permission$")
	regV1AgentsGetCustomerIDAddress = regexp.MustCompile("/v1/agents/get_by_customer_id_address$")

	// login
	regV1Login = regexp.MustCompile("/v1/login$")
)

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameAgentManager)

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
func NewListenHandler(sockHandler sockhandler.SockHandler, agentHandler agenthandler.AgentHandler) ListenHandler {
	h := &listenHandler{
		utilHandler: utilhandler.NewUtilHandler(),
		sockHandler: sockHandler,

		agentHandler: agentHandler,
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
		for {
			err := h.sockHandler.ConsumeRPC(queue, string(commonoutline.ServiceNameAgentManager), false, false, false, 10, h.processRequest)
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
	log := logrus.WithFields(logrus.Fields{
		"request": m,
	})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// agents
	////////////
	// GET /agents
	case regV1AgentsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AgentsGet(ctx, m)
		requestType = "/v1/agents"

	// POST /agents
	case regV1Agents.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AgentsPost(ctx, m)
		requestType = "/v1/agents"

	// GET /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AgentsIDGet(ctx, m)
		requestType = "/v1/agents/<agent-id>"

	// DELETE /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1AgentsIDDelete(ctx, m)
		requestType = "/v1/agents/<agent-id>"

	// PUT /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDPut(ctx, m)
		requestType = "/v1/agents/<agent-id>"

	// POST /agents/<username>/login
	case regV1AgentsUsernameLogin.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AgentsUsernameLogin(ctx, m)
		requestType = "/v1/agents/<agent-id>/login"

	// PUT /agents/<agent-id>/addresses
	case regV1AgentsIDAddresses.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDAddressesPut(ctx, m)
		requestType = "/v1/agents/<agent-id>/addresses"

	// PUT /agents/<agent-id>/password
	case regV1AgentsIDPassword.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDPasswordPut(ctx, m)
		requestType = "/v1/agents/<agent-id>/password"

	// PUT /agents/<agent-id>/tag_ids
	case regV1AgentsIDTagIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDTagIDsPut(ctx, m)
		requestType = "/v1/agents/<agent-id>/tag_ids"

	// PUT /agents/<agent-id>/status
	case regV1AgentsIDStatus.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDStatusPut(ctx, m)
		requestType = "/v1/agents/<agent-id>/status"

	// PUT /agents/<agent-id>/permission
	case regV1AgentsIDPermission.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AgentsIDPermissionPut(ctx, m)
		requestType = "/v1/agents/<agent-id>/permission"

	// POST /agents/get_by_customer_id_address
	case regV1AgentsGetCustomerIDAddress.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AgentsGetByCustomerIDAddressPost(ctx, m)
		requestType = "/v1/agents/get_by_customer_id_address"

	////////////
	// login
	////////////
	// POST /login
	case regV1Login.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1Login(ctx, m)
		requestType = "/v1/login"

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
		log.WithFields(logrus.Fields{
			"response": response,
		}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)
	}

	return response, err
}
