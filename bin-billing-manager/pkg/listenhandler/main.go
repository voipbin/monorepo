package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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
	rabbitSock rabbitmqhandler.Rabbit

	utilHandler    utilhandler.UtilHandler
	accountHandler accounthandler.AccountHandler
	billingHandler billinghandler.BillingHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// v1

	// accounts
	regV1Accounts                       = regexp.MustCompile("/v1/accounts$")
	regV1AccountsGet                    = regexp.MustCompile(`/v1/accounts\?`)
	regV1AccountsID                     = regexp.MustCompile("/v1/accounts/" + regUUID + "$")
	regV1AccountsIDBalanceAddForce      = regexp.MustCompile("/v1/accounts/" + regUUID + "/balance_add_force$")
	regV1AccountsIDBalanceSubtractForce = regexp.MustCompile("/v1/accounts/" + regUUID + "/balance_subtract_force$")
	regV1AccountsIDIsValidBalance       = regexp.MustCompile("/v1/accounts/" + regUUID + "/is_valid_balance$")
	regV1AccountsIDIsValidPaymentInfo   = regexp.MustCompile("/v1/accounts/" + regUUID + "/payment_info$")

	// billings
	regV1BillingsGet = regexp.MustCompile(`/v1/billings\?`)
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
func simpleResponse(code int) *rabbitmqhandler.Response {
	return &rabbitmqhandler.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	rabbitSock rabbitmqhandler.Rabbit,
	accountHandler accounthandler.AccountHandler,
	billingHandler billinghandler.BillingHandler,
) ListenHandler {
	h := &listenHandler{
		rabbitSock:     rabbitSock,
		utilHandler:    utilhandler.NewUtilHandler(),
		accountHandler: accountHandler,
		billingHandler: billingHandler,
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
			// consume the request
			err := h.rabbitSock.ConsumeRPCOpt(queue, constCosumerName, false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the request message correctly. err: %v", err)
			}
		}
	}()

	return nil
}

// processRequest handles all of requests of the listen queue.
func (h *listenHandler) processRequest(m *rabbitmqhandler.Request) (*rabbitmqhandler.Response, error) {

	var requestType string
	var err error
	var response *rabbitmqhandler.Response

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
	// GET /accounts
	case regV1AccountsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AccountsGet(ctx, m)
		requestType = "/v1/accounts"

	// POST /accounts
	case regV1Accounts.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AccountsPost(ctx, m)
		requestType = "/v1/accounts"

	// GET /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1AccountsIDGet(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// PUT /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AccountsIDPut(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// DELETE /accounts/<account-id>
	case regV1AccountsID.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodDelete:
		response, err = h.processV1AccountsIDDelete(ctx, m)
		requestType = "/v1/accounts/<account-id>"

	// POST /accounts/<account-id>/balance_add_force
	case regV1AccountsIDBalanceAddForce.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AccountsIDBalanceAddForcePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/balance_add_force"

	// POST /accounts/<account-id>/balance_subtract_force
	case regV1AccountsIDBalanceSubtractForce.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AccountsIDBalanceSubtractForcePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/balance_subtract_force"

	// POST /accounts/<account-id>/is_valid_balance
	case regV1AccountsIDIsValidBalance.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPost:
		response, err = h.processV1AccountsIDIsValidBalancePost(ctx, m)
		requestType = "/v1/accounts/<account-id>/is_valid_balance"

	// PUT /accounts/<account-id>/payment_info
	case regV1AccountsIDIsValidPaymentInfo.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodPut:
		response, err = h.processV1AccountsIDPaymentInfoPut(ctx, m)
		requestType = "/v1/accounts/<account-id>/payment_info"

	////////////////////
	// billings
	////////////////////
	// GET /billings
	case regV1BillingsGet.MatchString(m.URI) && m.Method == rabbitmqhandler.RequestMethodGet:
		response, err = h.processV1BillingsGet(ctx, m)
		requestType = "/v1/billings"

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
