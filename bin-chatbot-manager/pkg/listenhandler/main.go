package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chatbot-manager/pkg/chatbotcallhandler"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/messagehandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

// ListenHandler interface
type ListenHandler interface {
	Run() error
}

// listenHandler define
type listenHandler struct {
	sockHandler   sockhandler.SockHandler
	queueListen   string
	exchangeDelay string

	chatbotHandler     chatbothandler.ChatbotHandler
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler
	messageHandler     messagehandler.MessageHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	//// v1

	// chatbots
	regV1ChatbotsGet = regexp.MustCompile(`/v1/chatbots\?`)
	regV1Chatbots    = regexp.MustCompile("/v1/chatbots$")
	regV1ChatbotsID  = regexp.MustCompile("/v1/chatbots/" + regUUID + "$")

	// chatbotcalls
	regV1ChatbotcallsGet = regexp.MustCompile(`/v1/chatbotcalls\?`)
	regV1Chatbotcalls    = regexp.MustCompile(`/v1/chatbotcalls$`)
	regV1ChatbotcallsID  = regexp.MustCompile("/v1/chatbotcalls/" + regUUID + "$")

	// messages
	regV1MessagesGet = regexp.MustCompile(`/v1/messages\?`)
	regV1Messages    = regexp.MustCompile("/v1/messages$")
	regV1MessagesID  = regexp.MustCompile("/v1/messages/" + regUUID + "$")

	// service
	regV1ServicesTypeChatbotcall = regexp.MustCompile("/v1/services/type/chatbotcall$")
)

var (
	metricsNamespace = "chatbot_manager"

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

// getFilters parses the query and returns filters
func getFilters(u *url.URL) map[string]string {
	res := map[string]string{}

	keys := make([]string, 0, len(u.Query()))
	for k := range u.Query() {
		keys = append(keys, k)
	}

	for _, k := range keys {
		if strings.HasPrefix(k, "filter_") {
			tmp, _ := strings.CutPrefix(k, "filter_")
			res[tmp] = u.Query().Get(k)
		}
	}

	return res
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	queueListen string,
	exchangeDelay string,
	chatbotHandler chatbothandler.ChatbotHandler,
	chatbotcallHandler chatbotcallhandler.ChatbotcallHandler,
	messageHandler messagehandler.MessageHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:        sockHandler,
		queueListen:        queueListen,
		exchangeDelay:      exchangeDelay,
		chatbotHandler:     chatbotHandler,
		chatbotcallHandler: chatbotcallHandler,
		messageHandler:     messageHandler,
	}

	return h
}

// Run runs the listenhandler
func (h *listenHandler) Run() error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Run",
	})
	log.Info("Run the listenhandler.")

	if err := h.sockHandler.QueueCreate(h.queueListen, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// process requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), h.queueListen, "chatbot-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "processRequest",
		"request": m,
	})

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()
	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// chatbots
	////////////
	// GET /chatbots
	case regV1ChatbotsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ChatbotsGet(ctx, m)
		requestType = "/v1/chatbotcalls"

	// POST /chatbots
	case regV1Chatbots.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ChatbotsPost(ctx, m)
		requestType = "/v1/chatbotcalls"

	// GET /chatbots/<chatbot-id>
	case regV1ChatbotsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ChatbotsIDGet(ctx, m)
		requestType = "/v1/chatbots/<chatbot-id>"

	// DELETE /chatbots/<chatbot-id>
	case regV1ChatbotsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ChatbotsIDDelete(ctx, m)
		requestType = "/v1/chatbots/<chatbot-id>"

	// PUT /chatbots/<chatbot-id>
	case regV1ChatbotsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ChatbotsIDPut(ctx, m)
		requestType = "/v1/chatbots/<chatbot-id>"

	///////////////
	// chatbotcalls
	///////////////
	// GET /chatbotcalls
	case regV1ChatbotcallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ChatbotcallsGet(ctx, m)
		requestType = "/v1/chatbotcalls"

	// POST /chatbots
	case regV1Chatbotcalls.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ChatbotcallsPost(ctx, m)
		requestType = "/v1/chatbotcalls"

	// GET /chatbotcalls/<chatbotcall-id>
	case regV1ChatbotcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ChatbotcallsIDGet(ctx, m)
		requestType = "/v1/chatbotcalls/<chatbotcall-id>"

	// DELETE /chatbotcalls/<chatbotcall-id>
	case regV1ChatbotcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ChatbotcallsIDDelete(ctx, m)
		requestType = "/v1/chatbotcalls/<chatbotcall-id>"

	///////////////
	// messages
	///////////////
	// GET /messages
	case regV1MessagesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1MessagesGet(ctx, m)
		requestType = "/v1/messages"

	// POST /messages
	case regV1Messages.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1MessagesPost(ctx, m)
		requestType = "/v1/messages"

	// POST /messages/<message-id>
	case regV1MessagesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1MessagesIDGet(ctx, m)
		requestType = "/v1/messages/<message-id>"

	/////////////////
	// services
	////////////////
	// POST
	case regV1ServicesTypeChatbotcall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeChatbotcallPost(ctx, m)
		requestType = "/v1/services/type/chatbotcall"

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
		log.Errorf("Could not handle the requested message correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	return response, err
}
