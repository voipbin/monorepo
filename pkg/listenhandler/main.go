package listenhandler

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/contacthandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/trunkhandler"
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

	reqHandler       requesthandler.RequestHandler
	utilHandler      utilhandler.UtilHandler
	domainHandler    domainhandler.DomainHandler
	trunkHandler     trunkhandler.TrunkHandler
	extensionHandler extensionhandler.ExtensionHandler
	contactHandler   contacthandler.ContactHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1
	// contacts
	regV1ContactsGet = regexp.MustCompile(`/v1/contacts\?`)

	// domains
	regV1Domains           = regexp.MustCompile("/v1/domains$")
	regV1DomainsGet        = regexp.MustCompile(`/v1/domains\?`)
	regV1DomainsID         = regexp.MustCompile("/v1/domains/" + regUUID + "$")
	regV1DomainsDomainName = regexp.MustCompile("/v1/domains/domain_name/" + regAny)

	// extensions
	regV1Extensions    = regexp.MustCompile("/v1/extensions$")
	regV1ExtensionsGet = regexp.MustCompile(`/v1/extensions\?`)
	regV1ExtensionsID  = regexp.MustCompile("/v1/extensions/" + regUUID + "$")
	// regV1ExtensionsExtensionEndpoint     = regexp.MustCompile("/v1/extensions/endpoint/" + regAny + "$")
	regV1ExtensionsExtensionExtensionGet = regexp.MustCompile("/v1/extensions/extension/" + regAny + `\?`)

	// trunks
	regV1Trunks           = regexp.MustCompile("/v1/trunks$")
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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	reqHandler requesthandler.RequestHandler,
	domainHandler domainhandler.DomainHandler,
	trunkHandler trunkhandler.TrunkHandler,
	extensionHandler extensionhandler.ExtensionHandler,
	contactHandler contacthandler.ContactHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:       rabbitSock,
		reqHandler:       reqHandler,
		utilHandler:      utilhandler.NewUtilHandler(),
		domainHandler:    domainHandler,
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
			err := h.rabbitSock.ConsumeRPCOpt(queue, "registrar-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("could not consume the request message correctly. err: %v", err)
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

	log := logrus.WithFields(logrus.Fields{
		"func":      "processRequest",
		"uri":       m.URI,
		"method":    m.Method,
		"data_type": m.DataType,
		"data":      m.Data,
	},
	)
	log.Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// contacts
	////////////
	case regV1ContactsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ContactsGet(ctx, m)
		requestType = "/v1/contacts"

	case regV1ContactsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1ContactsPut(ctx, m)
		requestType = "/v1/contacts"

	////////////
	// domains
	////////////
	case regV1DomainsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1DomainsIDGet(ctx, m)
		requestType = "/v1/domains/<domain-id>"

	case regV1DomainsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1DomainsIDPut(ctx, m)
		requestType = "/v1/domains/<domain-id>"

	case regV1DomainsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1DomainsIDDelete(ctx, m)
		requestType = "/v1/domains/<domain-id>"

	case regV1Domains.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1DomainsPost(ctx, m)
		requestType = "/v1/domains"

	case regV1DomainsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1DomainsGet(ctx, m)
		requestType = "/v1/domains"

	case regV1DomainsDomainName.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1DomainsDomainNameDomainNameGet(ctx, m)
		requestType = "/v1/domains/domain_name/<domain-name>"

	/////////////
	// extensions
	/////////////
	case regV1ExtensionsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ExtensionsIDGet(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1ExtensionsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1ExtensionsIDPut(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1ExtensionsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ExtensionsIDDelete(ctx, m)
		requestType = "/v1/extensions/<extension-id>"

	case regV1Extensions.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ExtensionsPost(ctx, m)
		requestType = "/v1/extensions"

	case regV1ExtensionsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ExtensionsGet(ctx, m)
		requestType = "/v1/extensions"

	// case regV1ExtensionsExtensionEndpoint.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
	// 	response, err = h.processV1ExtensionsExtensionEndpointGet(ctx, m)
	// 	requestType = "/v1/extensions/endpoint/<endpoint>"

	case regV1ExtensionsExtensionExtensionGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ExtensionsExtensionExtensionGet(ctx, m)
		requestType = "/v1/extensions/extension/<extension>"

	/////////////
	// trunks
	/////////////
	case regV1TrunksGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TrunksGet(ctx, m)
		requestType = "/v1/trunks"

	case regV1Trunks.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1TrunksPost(ctx, m)
		requestType = "/v1/trunks"

	case regV1TrunksID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1TrunksIDGet(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1TrunksIDPut(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1TrunksIDDelete(ctx, m)
		requestType = "/v1/trunks/<trunk-id>"

	case regV1TrunksDomainName.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
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
