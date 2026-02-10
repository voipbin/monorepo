package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-registrar-manager/pkg/contacthandler"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"
	"monorepo/bin-registrar-manager/pkg/trunkhandler"
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

	reqHandler       requesthandler.RequestHandler
	utilHandler      utilhandler.UtilHandler
	trunkHandler     trunkhandler.TrunkHandler
	extensionHandler extensionhandler.ExtensionHandler
	contactHandler   contacthandler.ContactHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1
	// contacts
	regV1ContactsGet = regexp.MustCompile(`/v1/contacts(\?.*)?$`)

	// extensions
	regV1ExtensionsCountByCustomer = regexp.MustCompile("/v1/extensions/count_by_customer$")
	regV1Extensions                = regexp.MustCompile("/v1/extensions$")
	regV1ExtensionsGet             = regexp.MustCompile(`/v1/extensions\?`)
	regV1ExtensionsID  = regexp.MustCompile("/v1/extensions/" + regUUID + "$")
	// regV1ExtensionsExtensionEndpoint     = regexp.MustCompile("/v1/extensions/endpoint/" + regAny + "$")
	regV1ExtensionsExtensionExtensionGet = regexp.MustCompile("/v1/extensions/extension/" + regAny + `(\?.*)?$`)

	// extension-directs
	regV1ExtensionDirectsGet = regexp.MustCompile(`/v1/extension-directs\?`)

	// trunks
	regV1TrunksCountByCustomer = regexp.MustCompile("/v1/trunks/count_by_customer$")
	regV1Trunks                = regexp.MustCompile("/v1/trunks$")
	regV1TrunksGet        = regexp.MustCompile(`/v1/trunks\?`)
	regV1TrunksID         = regexp.MustCompile("/v1/trunks/" + regUUID + "$")
	regV1TrunksDomainName = regexp.MustCompile("/v1/trunks/domain_name/" + regAny)
)

var (
	metricsNamespace = "registrar_manager"

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
func NewListenHandler(
	sockHandler sockhandler.SockHandler,
	reqHandler requesthandler.RequestHandler,
	trunkHandler trunkhandler.TrunkHandler,
	extensionHandler extensionhandler.ExtensionHandler,
	contactHandler contacthandler.ContactHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:      sockHandler,
		reqHandler:       reqHandler,
		utilHandler:      utilhandler.NewUtilHandler(),
		trunkHandler:     trunkHandler,
		extensionHandler: extensionHandler,
		contactHandler:   contactHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "registrar-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("could not consume the request message correctly. err: %v", errConsume)
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
		"func":      "processRequest",
		"uri":       m.URI,
		"method":    m.Method,
		"data_type": m.DataType,
		"data":      m.Data,
	},
	)
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// contacts
	////////////
	case regV1ContactsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactsGet(ctx, m)
		requestType = "/v1/contacts"

	case regV1ContactsGet.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ContactsPut(ctx, m)
		requestType = "/v1/contacts"

	/////////////
	// extensions
	/////////////
	case regV1ExtensionsCountByCustomer.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionsCountByCustomerGet(ctx, m)
		requestType = "/v1/extensions/count_by_customer"

	case regV1ExtensionsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionsIDGet(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1ExtensionsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ExtensionsIDPut(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1ExtensionsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ExtensionsIDDelete(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1Extensions.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ExtensionsPost(ctx, m)
		requestType = "/v1/extensions"

	case regV1ExtensionsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionsGet(ctx, m)
		requestType = "/v1/extensions"

	// case regV1ExtensionsExtensionEndpoint.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
	// 	response, err = h.processV1ExtensionsExtensionEndpointGet(ctx, m)
	// 	requestType = "/v1/extensions/endpoint/<endpoint>"

	case regV1ExtensionsExtensionExtensionGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionsExtensionExtensionGet(ctx, m)
		requestType = "/v1/extensions/extension/<extension>"

	/////////////////
	// extension-directs
	/////////////////
	case regV1ExtensionDirectsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ExtensionDirectsGet(ctx, m)
		requestType = "/v1/extension-directs"

	/////////////
	// trunks
	/////////////
	case regV1TrunksCountByCustomer.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TrunksCountByCustomerGet(ctx, m)
		requestType = "/v1/trunks/count_by_customer"

	case regV1TrunksGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TrunksGet(ctx, m)
		requestType = "/v1/trunks"

	case regV1Trunks.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1TrunksPost(ctx, m)
		requestType = "/v1/trunks"

	case regV1TrunksID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TrunksIDGet(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1TrunksIDPut(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1TrunksIDDelete(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksDomainName.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1TrunksDomainNameDomainNameGet(ctx, m)
		requestType = "/v1/trunks/domain_name/<domain-name>"

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

	// default error handler
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err,
		}).Errorf("Could not process the request correctly. method: %s, uri: %s", m.Method, m.URI)
		response = simpleResponse(400)
		err = nil
	}

	log.WithFields(logrus.Fields{
		"response": response,
	}).Debugf("Sending back the resulut. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
