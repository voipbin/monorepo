package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

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

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/summaryhandler"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
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

	aiHandler      aihandler.AIHandler
	aicallHandler  aicallhandler.AIcallHandler
	messageHandler messagehandler.MessageHandler
	summaryHandler summaryhandler.SummaryHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	//// v1

	// ais
	regV1AIsGet = regexp.MustCompile(`/v1/ais\?`)
	regV1AIs    = regexp.MustCompile("/v1/ais$")
	regV1AIsID  = regexp.MustCompile("/v1/ais/" + regUUID + "$")

	// aicalls
	regV1AIcallsGet           = regexp.MustCompile(`/v1/aicalls\?`)
	regV1AIcalls              = regexp.MustCompile(`/v1/aicalls$`)
	regV1AIcallsID            = regexp.MustCompile("/v1/aicalls/" + regUUID + "$")
	regV1AIcallsIDTerminate   = regexp.MustCompile("/v1/aicalls/" + regUUID + "/terminate$")
	regV1AIcallsIDToolExecute = regexp.MustCompile("/v1/aicalls/" + regUUID + "/tool_execute$")

	// messages
	regV1MessagesGet = regexp.MustCompile(`/v1/messages\?`)
	regV1Messages    = regexp.MustCompile("/v1/messages$")
	regV1MessagesID  = regexp.MustCompile("/v1/messages/" + regUUID + "$")

	// service
	regV1ServicesTypeAIcall  = regexp.MustCompile("/v1/services/type/aicall$")
	regV1ServicesTypeSummary = regexp.MustCompile("/v1/services/type/summary$")
	regV1ServicesTypeTask    = regexp.MustCompile("/v1/services/type/task$")

	// summary
	regV1SummariesGet = regexp.MustCompile(`/v1/summaries\?`)
	regV1Summaries    = regexp.MustCompile("/v1/summaries$")
	regV1SummariesID  = regexp.MustCompile("/v1/summaries/" + regUUID + "$")
)

var (
	metricsNamespace = "ai_manager"

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

// convertToAIFilters converts string filters to ai.Field filters
func convertToAIFilters(filters map[string]string) map[ai.Field]any {
	res := make(map[ai.Field]any)
	for k, v := range filters {
		switch k {
		case "customer_id":
			res[ai.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "deleted":
			res[ai.FieldDeleted] = v == "true"
		default:
			res[ai.Field(k)] = v
		}
	}
	return res
}

// convertToAIcallFilters converts string filters to aicall.Field filters
func convertToAIcallFilters(filters map[string]string) map[aicall.Field]any {
	res := make(map[aicall.Field]any)
	for k, v := range filters {
		switch k {
		case "customer_id":
			res[aicall.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "ai_id":
			res[aicall.FieldAIID] = uuid.FromStringOrNil(v)
		case "activeflow_id":
			res[aicall.FieldActiveflowID] = uuid.FromStringOrNil(v)
		case "reference_id":
			res[aicall.FieldReferenceID] = uuid.FromStringOrNil(v)
		case "deleted":
			res[aicall.FieldDeleted] = v == "true"
		default:
			res[aicall.Field(k)] = v
		}
	}
	return res
}

// convertToMessageFilters converts string filters to message.Field filters
func convertToMessageFilters(filters map[string]string) map[message.Field]any {
	res := make(map[message.Field]any)
	for k, v := range filters {
		switch k {
		case "customer_id":
			res[message.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "aicall_id":
			res[message.FieldAIcallID] = uuid.FromStringOrNil(v)
		case "deleted":
			res[message.FieldDeleted] = v == "true"
		default:
			res[message.Field(k)] = v
		}
	}
	return res
}

// convertToSummaryFilters converts string filters to summary.Field filters
func convertToSummaryFilters(filters map[string]string) map[summary.Field]any {
	res := make(map[summary.Field]any)
	for k, v := range filters {
		switch k {
		case "customer_id":
			res[summary.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "activeflow_id":
			res[summary.FieldActiveflowID] = uuid.FromStringOrNil(v)
		case "reference_id":
			res[summary.FieldReferenceID] = uuid.FromStringOrNil(v)
		case "deleted":
			res[summary.FieldDeleted] = v == "true"
		default:
			res[summary.Field(k)] = v
		}
	}
	return res
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	queueListen string,
	exchangeDelay string,

	aiHandler aihandler.AIHandler,
	aicallHandler aicallhandler.AIcallHandler,
	messageHandler messagehandler.MessageHandler,
	summaryHandler summaryhandler.SummaryHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:   sockHandler,
		queueListen:   queueListen,
		exchangeDelay: exchangeDelay,

		aiHandler:      aiHandler,
		aicallHandler:  aicallHandler,
		messageHandler: messageHandler,
		summaryHandler: summaryHandler,
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
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), h.queueListen, string(outline.ServiceNameAIManager), false, false, false, 10, h.processRequest); errConsume != nil {
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
	// ais
	////////////
	// GET /ais
	case regV1AIsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AIsGet(ctx, m)
		requestType = "/v1/ais"

	// POST /ais
	case regV1AIs.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AIsPost(ctx, m)
		requestType = "/v1/ais"

	// GET /ais/<ai-id>
	case regV1AIsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AIsIDGet(ctx, m)
		requestType = "/v1/ais/<ai-id>"

	// DELETE /ais/<ai-id>
	case regV1AIsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1AIsIDDelete(ctx, m)
		requestType = "/v1/ais/<ai-id>"

	// PUT /ais/<ai-id>
	case regV1AIsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AIsIDPut(ctx, m)
		requestType = "/v1/ais/<ai-id>"

	///////////////
	// aicalls
	///////////////
	// GET /aicalls
	case regV1AIcallsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AIcallsGet(ctx, m)
		requestType = "/v1/aicalls"

	// POST /aicalls
	case regV1AIcalls.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AIcallsPost(ctx, m)
		requestType = "/v1/aicalls"

	// GET /aicalls/<aicall-id>
	case regV1AIcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AIcallsIDGet(ctx, m)
		requestType = "/v1/aicalls/<aicall-id>"

	// DELETE /aicalls/<aicall-id>
	case regV1AIcallsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1AIcallsIDDelete(ctx, m)
		requestType = "/v1/aicalls/<aicall-id>"

	// POST /aicalls/<aicall-id>/terminate
	case regV1AIcallsIDTerminate.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AIcallsIDTerminatePost(ctx, m)
		requestType = "/v1/aicalls/<aicall-id>/terminate"

	// POST /aicalls/<aicall-id>/tool_execute
	case regV1AIcallsIDToolExecute.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AIcallsIDToolExecutePost(ctx, m)
		requestType = "/v1/aicalls/<aicall-id>/tool_execute"

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
	/////////////////
	// POST /services/type/aicall
	case regV1ServicesTypeAIcall.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeAIcallPost(ctx, m)
		requestType = "/v1/services/type/aicall"

	// POST /services/type/summary
	case regV1ServicesTypeSummary.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeSummaryPost(ctx, m)
		requestType = "/v1/services/type/summary"

	// POST /services/type/task
	case regV1ServicesTypeTask.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ServicesTypeTaskPost(ctx, m)
		requestType = "/v1/services/type/task"

	/////////////////
	// summaries
	/////////////////
	// GET /summaries
	case regV1SummariesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SummariesGet(ctx, m)
		requestType = "/v1/summaries"

	// POST /summaries
	case regV1Summaries.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1SummariesPost(ctx, m)
		requestType = "/v1/summaries"

	// GET /summaries/<summary-id>
	case regV1SummariesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1SummariesIDGet(ctx, m)
		requestType = "/v1/summaries/<summary-id>"

	// DELETE /summaries/<summary-id>
	case regV1SummariesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1SummariesIDDelete(ctx, m)
		requestType = "/v1/summaries/<summary-id>"

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
