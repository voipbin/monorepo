package listenhandler

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/contacthandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/domainhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/extensionhandler"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/requesthandler"
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
	domainHandler    domainhandler.DomainHandler
	extensionHandler extensionhandler.ExtensionHandler
	contactHandler   contacthandler.ContactHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"
	regAny  = "(.*)"

	// v1
	// contacts
	regV1Contacts = regexp.MustCompile("/v1/contacts")

	// domains
	regV1Domains   = regexp.MustCompile("/v1/domains")
	regV1DomainsID = regexp.MustCompile("/v1/domains/" + regUUID)

	// extensions
	regV1Extensions   = regexp.MustCompile("/v1/extensions")
	regV1ExtensionsID = regexp.MustCompile("/v1/extensions/" + regUUID)
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
	extensionHandler extensionhandler.ExtensionHandler,
	contactHandler contacthandler.ContactHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:       rabbitSock,
		reqHandler:       reqHandler,
		domainHandler:    domainHandler,
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
		return fmt.Errorf("Could not declare the exchange for dealyed message. err: %v", err)
	}

	// bind a queue with delayed exchange
	if err := h.rabbitSock.QueueBind(queue, queue, exchangeDelay, false, nil); err != nil {
		return fmt.Errorf("Could not bind the queue and exchange. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPCOpt(queue, "registrar-manager", false, false, false, h.processRequest)
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

	uri, err := url.QueryUnescape(m.URI)
	if err != nil {
		uri = "could not unescape uri"
	}

	logrus.WithFields(
		logrus.Fields{
			"uri":       m.URI,
			"method":    m.Method,
			"data_type": m.DataType,
			"data":      m.Data,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, uri)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	////////////
	// contacts
	////////////
	case regV1Contacts.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ContactsGet(m)
		requestType = "/v1/contacts"

	////////////
	// domains
	////////////
	case regV1DomainsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1DomainsIDGet(m)
		requestType = "/v1/domains"

	case regV1DomainsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1DomainsIDPut(m)
		requestType = "/v1/domains"

	case regV1DomainsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1DomainsIDDelete(m)
		requestType = "/v1/domains"

	case regV1Domains.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1DomainsPost(m)
		requestType = "/v1/domains"

	case regV1Domains.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1DomainsGet(m)
		requestType = "/v1/domains"

	/////////////
	// extensions
	/////////////
	case regV1ExtensionsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ExtensionsIDGet(m)
		requestType = "/v1/extensions"

	case regV1ExtensionsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1ExtensionsIDPut(m)
		requestType = "/v1/extensions"

	case regV1ExtensionsID.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1ExtensionsIDDelete(m)
		requestType = "/v1/extensions"

	case regV1Extensions.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1ExtensionsPost(m)
		requestType = "/v1/extensions"

	case regV1Extensions.MatchString(m.URI) == true && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1ExtensionsGet(m)
		requestType = "/v1/extensions"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// No handler found
	/////////////////////////////////////////////////////////////////////////////////////////////////
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

	return response, err
}
