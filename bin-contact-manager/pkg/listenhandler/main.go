package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
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
	addressHandler addresshandler.AddressHandler
	caseHandler    casehandler.CaseHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1 contacts
	regV1Contacts       = regexp.MustCompile("/v1/contacts$")
	regV1ContactsGet    = regexp.MustCompile(`/v1/contacts\?(.*)$`)
	regV1ContactsID     = regexp.MustCompile("/v1/contacts/" + regUUID + "$")
	regV1ContactsLookup = regexp.MustCompile(`/v1/contacts/lookup\?(.*)$`)

	// v1 contact_addresses (independent resource)
	regV1ContactAddresses        = regexp.MustCompile("/v1/contact_addresses$")
	regV1ContactAddressesGet     = regexp.MustCompile(`/v1/contact_addresses\?(.*)$`)
	regV1ContactAddressesID      = regexp.MustCompile("/v1/contact_addresses/" + regUUID + `(\?.*)?$`)
	regV1ContactAddressesIDClaim = regexp.MustCompile("/v1/contact_addresses/" + regUUID + `/claim(\?.*)?$`)

	// v1 contacts/{id}/addresses
	regV1ContactsAddresses   = regexp.MustCompile("/v1/contacts/" + regUUID + "/addresses$")
	regV1ContactsAddressesID = regexp.MustCompile("/v1/contacts/" + regUUID + "/addresses/" + regUUID + "$")

	// v1 contacts/{id}/tags
	regV1ContactsTags   = regexp.MustCompile("/v1/contacts/" + regUUID + "/tags$")
	regV1ContactsTagsID = regexp.MustCompile("/v1/contacts/" + regUUID + "/tags/" + regUUID + "$")

	// v1 interactions
	regV1InteractionsUnresolved    = regexp.MustCompile(`/v1/interactions/unresolved(\?.*)?$`)
	regV1InteractionsGet           = regexp.MustCompile(`/v1/interactions\?(.*)$`)
	regV1InteractionsID            = regexp.MustCompile("/v1/interactions/" + regUUID + "$")
	regV1InteractionsResolutions   = regexp.MustCompile("/v1/interactions/" + regUUID + "/resolutions$")
	regV1InteractionsResolutionsID = regexp.MustCompile("/v1/interactions/" + regUUID + "/resolutions/" + regUUID + "$")

	// v1 cases
	regV1CasesUnresolved = regexp.MustCompile(`/v1/cases/unresolved(\?.*)?$`)
	regV1Cases           = regexp.MustCompile(`/v1/cases$`)
	regV1CasesGet        = regexp.MustCompile(`/v1/cases\?(.*)$`)
	regV1CasesID         = regexp.MustCompile("/v1/cases/" + regUUID + "$")
	regV1CasesIDClose    = regexp.MustCompile("/v1/cases/" + regUUID + "/close$")
	regV1CasesIDAssign   = regexp.MustCompile("/v1/cases/" + regUUID + "/assign$")
	regV1CasesIDContinue = regexp.MustCompile("/v1/cases/" + regUUID + "/continue$")
	regV1CasesIDNotes    = regexp.MustCompile("/v1/cases/" + regUUID + "/notes$")
	regV1CasesIDNotesID  = regexp.MustCompile("/v1/cases/" + regUUID + "/notes/" + regUUID + "$")
	regV1CasesIDTags     = regexp.MustCompile("/v1/cases/" + regUUID + "/tags$")
	regV1CasesIDTagsID   = regexp.MustCompile("/v1/cases/" + regUUID + "/tags/" + regUUID + "$")
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

// errorResponse maps a business-handler error to the appropriate sock.Response.
// Resolution order: typed *cerrors.VoipbinError → ToResponse; legacy
// dbhandler.ErrNotFound → 404; else → 500.
func errorResponse(err error) *sock.Response {
	if err == nil {
		logrus.WithField("func", "errorResponse").Warn("errorResponse called with nil error — likely a caller bug; returning 500")
		return simpleResponse(http.StatusInternalServerError)
	}

	var ve *cerrors.VoipbinError
	if stderrors.As(err, &ve) {
		resp, e := cerrors.ToResponse(ve)
		if e == nil {
			return resp
		}
		logrus.WithField("func", "errorResponse").Errorf("cerrors.ToResponse failed for typed VoipbinError: %v", e)
		return simpleResponse(http.StatusInternalServerError)
	}

	if stderrors.Is(err, dbhandler.ErrNotFound) {
		return simpleResponse(http.StatusNotFound)
	}

	return simpleResponse(http.StatusInternalServerError)
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(sockHandler sockhandler.SockHandler, contactHandler contacthandler.ContactHandler, addressHandler addresshandler.AddressHandler, caseHandler casehandler.CaseHandler) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		utilHandler:    utilhandler.NewUtilHandler(),
		contactHandler: contactHandler,
		addressHandler: addressHandler,
		caseHandler:    caseHandler,
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
	// v1 Contact Addresses (independent resource)
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// GET /contact_addresses?...
	case regV1ContactAddressesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactAddressesGet(ctx, m)
		requestType = "/v1/contact_addresses"

	// POST /contact_addresses
	case regV1ContactAddresses.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactAddressesPost(ctx, m)
		requestType = "/v1/contact_addresses"

	// GET /contact_addresses/{id}
	case regV1ContactAddressesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactAddressesIDGet(ctx, m)
		requestType = "/v1/contact_addresses/{id}"

	// PUT /contact_addresses/{id}
	case regV1ContactAddressesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ContactAddressesIDPut(ctx, m)
		requestType = "/v1/contact_addresses/{id}"

	// DELETE /contact_addresses/{id}
	case regV1ContactAddressesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactAddressesIDDelete(ctx, m)
		requestType = "/v1/contact_addresses/{id}"

	// POST /contact_addresses/{id}/claim
	case regV1ContactAddressesIDClaim.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactAddressesIDClaim(ctx, m)
		requestType = "/v1/contact_addresses/{id}/claim"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Contacts Addresses
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// GET /contacts/{id}/addresses
	case regV1ContactsAddresses.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1ContactsAddressesGet(ctx, m)
		requestType = "/v1/contacts/{id}/addresses"

	// POST /contacts/{id}/addresses
	case regV1ContactsAddresses.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1ContactsAddressesPost(ctx, m)
		requestType = "/v1/contacts/{id}/addresses"

	// PUT /contacts/{id}/addresses/{address_id}
	case regV1ContactsAddressesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1ContactsAddressesIDPut(ctx, m)
		requestType = "/v1/contacts/{id}/addresses/{address_id}"

	// DELETE /contacts/{id}/addresses/{address_id}
	case regV1ContactsAddressesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1ContactsAddressesIDDelete(ctx, m)
		requestType = "/v1/contacts/{id}/addresses/{address_id}"

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
	// v1 Interactions
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// GET /interactions/unresolved (must be before regV1InteractionsID)
	case regV1InteractionsUnresolved.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1InteractionsUnresolvedGet(ctx, m)
		requestType = "/v1/interactions/unresolved"

	// GET /interactions?...
	case regV1InteractionsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1InteractionsGet(ctx, m)
		requestType = "/v1/interactions"

	// GET /interactions/{id}
	case regV1InteractionsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1InteractionsIDGet(ctx, m)
		requestType = "/v1/interactions/{id}"

	// POST /interactions/{id}/resolutions
	case regV1InteractionsResolutions.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1InteractionsResolutionsPost(ctx, m)
		requestType = "/v1/interactions/{id}/resolutions"

	// DELETE /interactions/{id}/resolutions/{rid}
	case regV1InteractionsResolutionsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1InteractionsResolutionsIDDelete(ctx, m)
		requestType = "/v1/interactions/{id}/resolutions/{rid}"

	/////////////////////////////////////////////////////////////////////////////////////////////////
	// v1 Cases
	/////////////////////////////////////////////////////////////////////////////////////////////////

	// GET /cases/unresolved (must be before regV1CasesID)
	case regV1CasesUnresolved.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CasesUnresolvedGet(ctx, m)
		requestType = "/v1/cases/unresolved"

	// GET /cases?...
	case regV1CasesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CasesGet(ctx, m)
		requestType = "/v1/cases"

	// POST /cases
	case regV1Cases.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesPost(ctx, m)
		requestType = "/v1/cases"

	// GET /cases/{id}
	case regV1CasesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CasesIDGet(ctx, m)
		requestType = "/v1/cases/{id}"

	// PUT /cases/{id}
	case regV1CasesID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CasesIDPut(ctx, m)
		requestType = "/v1/cases/{id}"

	// POST /cases/{id}/close
	case regV1CasesIDClose.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesIDClosePost(ctx, m)
		requestType = "/v1/cases/{id}/close"

	// POST /cases/{id}/assign
	case regV1CasesIDAssign.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesIDAssignPost(ctx, m)
		requestType = "/v1/cases/{id}/assign"

	// POST /cases/{id}/continue
	case regV1CasesIDContinue.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesIDContinuePost(ctx, m)
		requestType = "/v1/cases/{id}/continue"

	// GET /cases/{id}/notes
	case regV1CasesIDNotes.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CasesIDNotesGet(ctx, m)
		requestType = "/v1/cases/{id}/notes"

	// POST /cases/{id}/notes
	case regV1CasesIDNotes.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesIDNotesPost(ctx, m)
		requestType = "/v1/cases/{id}/notes"

	// DELETE /cases/{id}/notes/{note_id}
	case regV1CasesIDNotesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CasesIDNotesIDDelete(ctx, m)
		requestType = "/v1/cases/{id}/notes/{note_id}"

	// GET /cases/{id}/tags
	case regV1CasesIDTags.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CasesIDTagsGet(ctx, m)
		requestType = "/v1/cases/{id}/tags"

	// POST /cases/{id}/tags
	case regV1CasesIDTags.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CasesIDTagsPost(ctx, m)
		requestType = "/v1/cases/{id}/tags"

	// DELETE /cases/{id}/tags/{tag_id}
	case regV1CasesIDTagsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CasesIDTagsIDDelete(ctx, m)
		requestType = "/v1/cases/{id}/tags/{tag_id}"

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

	// default error handler — typed errors and ErrNotFound flow through
	// errorResponse; other errors keep legacy 400.
	if err != nil {
		log.Errorf("Could not process request. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		var ve *cerrors.VoipbinError
		switch {
		case stderrors.As(err, &ve):
			response = errorResponse(err)
		case stderrors.Is(err, dbhandler.ErrNotFound):
			response = errorResponse(err)
		default:
			response = simpleResponse(400)
		}
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
