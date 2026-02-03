package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/contacthandler"
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

	utilHandler    utilhandler.UtilHandler
	contactHandler contacthandler.ContactHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1 contacts
	regV1Contacts       = regexp.MustCompile("/v1/contacts$")
	regV1ContactsGet    = regexp.MustCompile(`/v1/contacts\?(.*)$`)
	regV1ContactsID     = regexp.MustCompile("/v1/contacts/" + regUUID + "$")
	regV1ContactsLookup = regexp.MustCompile(`/v1/contacts/lookup\?(.*)$`)

	// v1 contacts/{id}/phone-numbers
	regV1ContactsPhoneNumbers   = regexp.MustCompile("/v1/contacts/" + regUUID + "/phone-numbers$")
	regV1ContactsPhoneNumbersID = regexp.MustCompile("/v1/contacts/" + regUUID + "/phone-numbers/" + regUUID + "$")

	// v1 contacts/{id}/emails
	regV1ContactsEmails   = regexp.MustCompile("/v1/contacts/" + regUUID + "/emails$")
	regV1ContactsEmailsID = regexp.MustCompile("/v1/contacts/" + regUUID + "/emails/" + regUUID + "$")

	// v1 contacts/{id}/tags
	regV1ContactsTags   = regexp.MustCompile("/v1/contacts/" + regUUID + "/tags$")
	regV1ContactsTagsID = regexp.MustCompile("/v1/contacts/" + regUUID + "/tags/" + regUUID + "$")
)

var (
	metricsNamespace = "contact_manager"

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
func NewListenHandler(sockHandler sockhandler.SockHandler, contactHandler contacthandler.ContactHandler) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		utilHandler:    utilhandler.NewUtilHandler(),
		contactHandler: contactHandler,
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
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "contact-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {
	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()
	log := logrus.WithFields(
		logrus.Fields{
			"request": m,
		})
	log.Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {
	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Contacts
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// GET /contacts/lookup?...
	case regV1ContactsLookup.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactsLookupGet(ctx, m)
		requestType = "/v1/contacts/lookup"

	// GET /contacts?...
	case regV1ContactsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactsGet(ctx, m)
		requestType = "/v1/contacts"

	// POST /contacts
	case regV1Contacts.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactsPost(ctx, m)
		requestType = "/v1/contacts"

	// GET /contacts/{id}
	case regV1ContactsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactsIDGet(ctx, m)
		requestType = "/v1/contacts/{id}"

	// PUT /contacts/{id}
	case regV1ContactsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ContactsIDPut(ctx, m)
		requestType = "/v1/contacts/{id}"

	// DELETE /contacts/{id}
	case regV1ContactsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactsIDDelete(ctx, m)
		requestType = "/v1/contacts/{id}"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Contacts Phone Numbers
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// POST /contacts/{id}/phone-numbers
	case regV1ContactsPhoneNumbers.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactsPhoneNumbersPost(ctx, m)
		requestType = "/v1/contacts/{id}/phone-numbers"

	// DELETE /contacts/{id}/phone-numbers/{phone_id}
	case regV1ContactsPhoneNumbersID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactsPhoneNumbersIDDelete(ctx, m)
		requestType = "/v1/contacts/{id}/phone-numbers/{phone_id}"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Contacts Emails
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// POST /contacts/{id}/emails
	case regV1ContactsEmails.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactsEmailsPost(ctx, m)
		requestType = "/v1/contacts/{id}/emails"

	// DELETE /contacts/{id}/emails/{email_id}
	case regV1ContactsEmailsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactsEmailsIDDelete(ctx, m)
		requestType = "/v1/contacts/{id}/emails/{email_id}"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Contacts Tags
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// POST /contacts/{id}/tags
	case regV1ContactsTags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactsTagsPost(ctx, m)
		requestType = "/v1/contacts/{id}/tags"

	// DELETE /contacts/{id}/tags/{tag_id}
	case regV1ContactsTagsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactsTagsIDDelete(ctx, m)
		requestType = "/v1/contacts/{id}/tags/{tag_id}"

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
		log.Errorf("Could not process request. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	} else {
		log.WithFields(
			logrus.Fields{
				"response": response,
			},
		).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)
	}

	return response, err
}
