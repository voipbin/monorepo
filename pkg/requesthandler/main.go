package requesthandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requesthandler -destination ./mock_requesthandler_requesthandler.go -source main.go -build_flags=-mod=mod

import (
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requesthandler/models/telnyx"
)

// contents type
var (
	ContentTypeText = "text/plain"
	ContentTypeJSON = "application/json"
)

// group asterisk id
var (
	AsteriskIDCall       = "call"       // asterisk-call
	AsteriskIDConference = "conference" // asterisk-conference
)

// delay units
const (
	DelayNow    int = 0
	DelaySecond int = 1000
	DelayMinute int = DelaySecond * 60
	DelayHour   int = DelayMinute * 60
)

// telnyx
const (
	TelnyxToken string = "KEY017B6ED1E90D8FC5DB6ED95F1ACFE4F5_WzTaTxsXJCdwOviG4t1xMM"
)

// twilio
const (
	TwilioSID   string = "AC3300cb9426b78c9ce48db86a755166f0"
	TwilioToken string = "58c603e14220f52553be7769b209f423"
)

var (
	metricsNamespace = "number_manager"

	promRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "request_process_time",
			Help:      "Process time of send/receiv requests",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"target", "resource", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promRequestProcessTime,
	)
}

// RequestHandler intreface for ARI request handler
type RequestHandler interface {

	// telnyx
	TelnyxAvailableNumberGets(countryCode, locality, administrativeArea string, limit uint) ([]*telnyx.AvailableNumber, error)

	TelnyxNumberOrdersPost(numbers []string) (*telnyx.OrderNumber, error)

	TelnyxPhoneNumbersGet(size uint, tag, number string) ([]*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDGet(id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDDelete(id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDUpdateConnectionID(id string, connectionID string) (*telnyx.PhoneNumber, error)
}

type requestHandler struct {
	sock rabbitmqhandler.Rabbit

	exchangeDelay string
}

// NewRequestHandler create RequesterHandler
func NewRequestHandler(sock rabbitmqhandler.Rabbit, exchangeDelay string) RequestHandler {
	h := &requestHandler{
		sock: sock,

		exchangeDelay: exchangeDelay,
	}

	return h
}
