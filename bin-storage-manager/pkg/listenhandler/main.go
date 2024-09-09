package listenhandler

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
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
	utilHandler    utilhandler.UtilHandler
	rabbitSock     rabbitmqhandler.Rabbit
	storageHandler storagehandler.StorageHandler
	accountHandler accounthandler.AccountHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// accounts
	regV1AccountsGet = regexp.MustCompile(`/v1/accounts\?`)
	regV1Accounts    = regexp.MustCompile("/v1/accounts$")
	regV1AccountsID  = regexp.MustCompile("/v1/accounts/" + regUUID + "$")

	// files
	regV1FilesGet = regexp.MustCompile(`/v1/files\?`)
	regV1Files    = regexp.MustCompile("/v1/files$")
	regV1FilesID  = regexp.MustCompile("/v1/files/" + regUUID + "$")

	// compress
	regV1Compressfiles = regexp.MustCompile("/v1/compressfiles$")

	// recordings
	regV1RecordingsID = regexp.MustCompile("/v1/recordings/(.*)")
)

var (
	metricsNamespace = "storage_manager"

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
	rabbitSock rabbitmqhandler.Rabbit,
	storageHandler storagehandler.StorageHandler,
	accountHandler accounthandler.AccountHandler,
) ListenHandler {
	h := &listenHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		rabbitSock:     rabbitSock,
		storageHandler: storageHandler,
		accountHandler: accountHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	logrus.WithFields(logrus.Fields{
		"queue": queue,
	}).Info("Creating rabbitmq queue for listen.")

	if err := h.rabbitSock.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// receive requests
	go func() {
		for {
			err := h.rabbitSock.ConsumeRPC(queue, "storage-manager", false, false, false, 10, h.processRequest)
			if err != nil {
				logrus.Errorf("Could not consume the message correctly. Will try again after 1 second. err: %v", err)
				time.Sleep(time.Second * 1)
			}
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

	logrus.WithFields(
		logrus.Fields{
			"request": m,
		}).Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {

	///////////////////////////////////////////////////////////////////////
	// v1 /////////////////////////////////////////////////////////////////
	///////////////////////////////////////////////////////////////////////

	// accounts /////////////
	case regV1Accounts.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/accounts"
		response, err = h.v1AccountsPost(ctx, m)

	case regV1AccountsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/accounts"
		response, err = h.v1AccountsGet(ctx, m)

	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/accounts/<account-id>"
		response, err = h.v1AccountsIDGet(ctx, m)

	case regV1AccountsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/accounts/<account-id>"
		response, err = h.v1AccountsIDDelete(ctx, m)

	// compressfiles
	case regV1Compressfiles.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/compressfiles"
		response, err = h.v1CompressfilesPost(ctx, m)

	// files ////////////////
	case regV1Files.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/files"
		response, err = h.v1FilesPost(ctx, m)

	case regV1FilesGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/files"
		response, err = h.v1FilesGet(ctx, m)

	case regV1FilesID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/files/<file-id>"
		response, err = h.v1FilesIDGet(ctx, m)

	case regV1FilesID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/files/<file-id>"
		response, err = h.v1FilesIDDelete(ctx, m)

	// recordings /////////////
	case regV1RecordingsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/recordings"
		response, err = h.v1RecordingsIDGet(ctx, m)

	case regV1RecordingsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/recordings"
		response, err = h.v1RecordingsIDDelete(ctx, m)

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

	logrus.WithFields(
		logrus.Fields{
			"request":  m,
			"response": response,
		}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}
