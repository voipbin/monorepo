package listenhandler

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

	"monorepo/bin-billing-manager/pkg/accounthandler"
	"monorepo/bin-billing-manager/pkg/billinghandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

const (
	constCosumerName = "billing-manager"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler

	utilHandler    utilhandler.UtilHandler
	accountHandler accounthandler.AccountHandler
	billingHandler billinghandler.BillingHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// accounts
	regV1AccountsID                     = regexp.MustCompile("/v1/accounts/" + regUUID + "$")
	regV1AccountsIDBalanceAddForce      = regexp.MustCompile("/v1/accounts/" + regUUID + "/balance_add_force$")
	regV1AccountsIDBalanceSubtractForce = regexp.MustCompile("/v1/accounts/" + regUUID + "/balance_subtract_force$")
	regV1AccountsIDIsValidBalance        = regexp.MustCompile("/v1/accounts/" + regUUID + "/is_valid_balance$")
	regV1AccountsIDIsValidResourceLimit = regexp.MustCompile("/v1/accounts/" + regUUID + "/is_valid_resource_limit$")
	regV1AccountsIDIsValidPaymentInfo   = regexp.MustCompile("/v1/accounts/" + regUUID + "/payment_info$")

	regV1AccountsIsValidBalanceByCustomerID       = regexp.MustCompile("/v1/accounts/is_valid_balance_by_customer_id$")
	regV1AccountsIsValidResourceLimitByCustomerID = regexp.MustCompile("/v1/accounts/is_valid_resource_limit_by_customer_id$")

	// billings
	regV1BillingsGet = regexp.MustCompile(`/v1/billings\?`)
	regV1BillingGet  = regexp.MustCompile("/v1/billings/" + regUUID + "$")
)

var (
	metricsNamespace = "billing_manager"

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
	accountHandler accounthandler.AccountHandler,
	billingHandler billinghandler.BillingHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler:    sockHandler,
		utilHandler:    utilhandler.NewUtilHandler(),
		accountHandler: accountHandler,
		billingHandler: billingHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"queue": queue,
	})
	log.Info("Creating rabbitmq queue for listen.")

	// declare the queue
	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, constCosumerName, false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
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

	////////////////////
	// accounts
	////////////////////
	// GET /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1AccountsIDGet(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// PUT /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AccountsIDPut(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// POST /accounts/<account-id>/balance_add_force
	case regV1AccountsIDBalanceAddForce.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIDBalanceAddForcePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/balance_add_force"

	// POST /accounts/<account-id>/balance_subtract_force
	case regV1AccountsIDBalanceSubtractForce.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIDBalanceSubtractForcePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/balance_subtract_force"

	// POST /accounts/<account-id>/is_valid_balance
	case regV1AccountsIDIsValidBalance.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIDIsValidBalancePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/is_valid_balance"

	// POST /accounts/<account-id>/is_valid_resource_limit
	case regV1AccountsIDIsValidResourceLimit.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIDIsValidResourceLimitPost(ctx, m)
		requestType = "/v1/accounts/<account-id>/is_valid_resource_limit"

	// PUT /accounts/<account-id>/payment_info
	case regV1AccountsIDIsValidPaymentInfo.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		response, err = h.processV1AccountsIDPaymentInfoPut(ctx, m)
		requestType = "/v1/accounts/<account-id>/payment_info"

	// POST /accounts/is_valid_balance_by_customer_id
	case regV1AccountsIsValidBalanceByCustomerID.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIsValidBalanceByCustomerIDPost(ctx, m)
		requestType = "/v1/accounts/is_valid_balance_by_customer_id"

	// POST /accounts/is_valid_resource_limit_by_customer_id
	case regV1AccountsIsValidResourceLimitByCustomerID.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		response, err = h.processV1AccountsIsValidResourceLimitByCustomerIDPost(ctx, m)
		requestType = "/v1/accounts/is_valid_resource_limit_by_customer_id"

	////////////////////
	// billings
	////////////////////
	// GET /billings
	case regV1BillingsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1BillingsGet(ctx, m)
		requestType = "/v1/billings"

	// GET /billings/<billing-id>
	case regV1BillingGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		response, err = h.processV1BillingGet(ctx, m)
		requestType = "/v1/billing"

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
		log.Errorf("Could not process the request correctly. method: %s, uri: %s, err: %v", m.Method, m.URI, err)
		response = simpleResponse(400)
		err = nil
	}

	log.WithFields(logrus.Fields{
		"response": response,
	}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
