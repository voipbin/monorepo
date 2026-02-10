package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/sirupsen/logrus"

	"monorepo/bin-customer-manager/pkg/accesskeyhandler"
	"monorepo/bin-customer-manager/pkg/customerhandler"
	"monorepo/bin-customer-manager/pkg/metricshandler"
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
	customerHandler  customerhandler.CustomerHandler
	accesskeyHandler accesskeyhandler.AccesskeyHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}" //nolint:deadcode,unused,varcheck // this is ok

	// v1

	// accesskeys
	regV1Accesskeys    = regexp.MustCompile("/v1/accesskeys$")
	regV1AccesskeysGet = regexp.MustCompile(`/v1/accesskeys\?(.*)$`)
	regV1AccesskeysID  = regexp.MustCompile("/v1/accesskeys/" + regUUID + "$")

	// customers
	regV1Customers                     = regexp.MustCompile("/v1/customers$")
	regV1CustomersGet                  = regexp.MustCompile(`/v1/customers\?(.*)$`)
	regV1CustomersID                   = regexp.MustCompile("/v1/customers/" + regUUID + "$")
	regV1CustomersIDIsValidBalance        = regexp.MustCompile("/v1/customers/" + regUUID + "/is_valid_balance$")
	regV1CustomersIDIsValidResourceLimit = regexp.MustCompile("/v1/customers/" + regUUID + "/is_valid_resource_limit$")
	regV1CustomersIDIsBillingAccountID   = regexp.MustCompile("/v1/customers/" + regUUID + "/billing_account_id$")

	regV1CustomersSignup      = regexp.MustCompile("/v1/customers/signup$")
	regV1CustomersEmailVerify = regexp.MustCompile("/v1/customers/email_verify$")
)


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
	customerHandler customerhandler.CustomerHandler,
	accesskeyHandler accesskeyhandler.AccesskeyHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:      sockHandler,
		reqHandler:       reqHandler,
		utilHandler:      utilhandler.NewUtilHandler(),
		customerHandler:  customerHandler,
		accesskeyHandler: accesskeyHandler,
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
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, "call-manager", false, false, false, 10, h.processRequest); errConsume != nil {
			logrus.Errorf("Could not consume the request message correctly. err: %v", errConsume)
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
	// v1
	/////////////////////////////////////////////////////////////////////////////////////////////////

	//////////////
	// accesskeys
	//////////////
	// GET /accesskeys
	case regV1AccesskeysGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AccesskeysGet(ctx, m)
		requestType = "/v1/accesskeys"

	// POST /accesskeys
	case regV1Accesskeys.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccesskeysPost(ctx, m)
		requestType = "/v1/accesskeys"

	// GET /accesskeys/<accesskey-id>
	case regV1AccesskeysID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AccesskeysIDGet(ctx, m)
		requestType = "/v1/accesskeys/<accesskey-id>"

	// PUT /accesskeys/<accesskey-id>
	case regV1AccesskeysID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AccesskeysIDPut(ctx, m)
		requestType = "/v1/accesskeys/<accesskey-id>"

	// DELETE /accesskeys/<accesskey-id>
	case regV1AccesskeysID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1AccesskeysIDDelete(ctx, m)
		requestType = "/v1/accesskeys/<accesskey-id>"

	////////////
	// customers
	////////////
	// POST /customers/signup
	case regV1CustomersSignup.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersSignupPost(ctx, m)
		requestType = "/v1/customers/signup"

	// POST /customers/email_verify
	case regV1CustomersEmailVerify.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersEmailVerifyPost(ctx, m)
		requestType = "/v1/customers/email_verify"

	// GET /customers
	case regV1CustomersGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CustomersGet(ctx, m)
		requestType = "/v1/customers"

	// POST /customers
	case regV1Customers.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersPost(ctx, m)
		requestType = "/v1/customers"

	// GET /customers/<customer-id>
	case regV1CustomersID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1CustomersIDGet(ctx, m)
		requestType = "/v1/customers"

	// PUT /customers/<customer-id>
	case regV1CustomersID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CustomersIDPut(ctx, m)
		requestType = "/v1/customers"

	// DELETE /customers/<customer-id>
	case regV1CustomersID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		response, err = h.processV1CustomersIDDelete(ctx, m)
		requestType = "/v1/customers"

	// PUT /customers/<customer-id>/billing_account_id
	case regV1CustomersIDIsBillingAccountID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1CustomersIDBillingAccountIDPut(ctx, m)
		requestType = "/v1/customers/<customer_id>/billing_account_id"

	// POST /customers/<customer-id>/is_valid_balance
	case regV1CustomersIDIsValidBalance.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersIDIsValidBalance(ctx, m)
		requestType = "/v1/customers/<customer_id>/is_valid_balance"

	// POST /customers/<customer-id>/is_valid_resource_limit
	case regV1CustomersIDIsValidResourceLimit.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1CustomersIDIsValidResourceLimit(ctx, m)
		requestType = "/v1/customers/<customer_id>/is_valid_resource_limit"

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
	metricshandler.ReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	if err != nil {
		log.Errorf("Could not find corresponded message handler. method: %s, uri: %s", m.Method, m.URI)
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
