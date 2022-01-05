package listenhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package listenhandler -destination ./mock_listenhandler_listenhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/agenthandler"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/taghandler"
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

	agentHandler agenthandler.AgentHandler
	tagHandler   taghandler.TagHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok
	regAny  = "(.*)"

	// v1
	// agents
	regV1Agents              = regexp.MustCompile("/v1/agents$")
	regV1AgentsGet           = regexp.MustCompile(`/v1/agents\?(.*)$`)
	regV1AgentsUsernameLogin = regexp.MustCompile("/v1/agents/" + regAny + "/login$")
	regV1AgentsID            = regexp.MustCompile("/v1/agents/" + regUUID + "$")
	regV1AgentsIDAddresses   = regexp.MustCompile("/v1/agents/" + regUUID + "/addresses$")
	regV1AgentsIDTagIDs      = regexp.MustCompile("/v1/agents/" + regUUID + "/tag_ids$")
	regV1AgentsIDStatus      = regexp.MustCompile("/v1/agents/" + regUUID + "/status$")
	regV1AgentsIDPassword    = regexp.MustCompile("/v1/agents/" + regUUID + "/password$")
	regV1AgentsIDDial        = regexp.MustCompile("/v1/agents/" + regUUID + "/dial$")

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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(rabbitSock rabbitmqhandler.Rabbit, agentHandler agenthandler.AgentHandler, tagHandler taghandler.TagHandler) ListenHandler {
	h := &listenHandler{
		rabbitSock: rabbitSock,

		agentHandler: agentHandler,
		tagHandler:   tagHandler,
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
		return fmt.Errorf("could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("could not bind the queue and exchange. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPCOpt(queue, "call-manager", false, false, false, h.processRequest)
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

	ctx := context.Background()

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}

	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// agents
	////////////
	// GET /agents
	case regV1AgentsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AgentsGet(ctx, m)
		requestType = "/v1/agents"

	// POST /agents
	case regV1Agents.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AgentsPost(ctx, m)
		requestType = "/v1/agents"

	// GET /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AgentsIDGet(ctx, m)
		requestType = "/v1/agents"

	// DELETE /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1AgentsIDDelete(ctx, m)
		requestType = "/v1/agents"

	// PUT /agents/<agent-id>
	case regV1AgentsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AgentsIDPut(ctx, m)
		requestType = "/v1/agents"

	// POST /agents/<username>/login
	case regV1AgentsUsernameLogin.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AgentsUsernameLogin(ctx, m)
		requestType = "/v1/agents"

	// PUT /agents/<agent-id>/addresses
	case regV1AgentsIDAddresses.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AgentsIDAddressesPut(ctx, m)
		requestType = "/v1/agents"

	// PUT /agents/<agent-id>/password
	case regV1AgentsIDPassword.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AgentsIDPasswordPut(ctx, m)
		requestType = "/v1/agents"

	// PUT /agents/<agent-id>/tag_ids
	case regV1AgentsIDTagIDs.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AgentsIDTagIDsPut(ctx, m)
		requestType = "/v1/agents"

	// PUT /agents/<agent-id>/tag_ids
	case regV1AgentsIDStatus.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AgentsIDStatusPut(ctx, m)
		requestType = "/v1/agents"

	// POST /agents/<agent-id>/dial
	case regV1AgentsIDDial.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AgentsIDDialPost(ctx, m)
		requestType = "/v1/agents"

	////////////
	// tags
	////////////
	// GET /tags
	case regV1TagsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TagsGet(ctx, m)
		requestType = "/v1/tags"

	// POST /tags
	case regV1Tags.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1TagsPost(ctx, m)
		requestType = "/v1/tags"

	// DELETE /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1TagsIDDelete(ctx, m)
		requestType = "/v1/tags"

	// GET /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TagsIDGet(ctx, m)
		requestType = "/v1/tags"

	// PUT /tags/<tag-id>
	case regV1TagsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1TagsIDPut(ctx, m)
		requestType = "/v1/tags"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
	default:
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, uri)
		response = simpleResponse(400)
		err = nil
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, uri)
	}

	return response, err
}

func parseTagIDs(t string) []uuid.UUID {
	str := strings.Split(t, ",")

	res := []uuid.UUID{}
	for _, s := range str {
		tmp := uuid.FromStringOrNil(s)
		if tmp == uuid.Nil {
			continue
		}

		res = append(res, tmp)
	}

	return res
}
