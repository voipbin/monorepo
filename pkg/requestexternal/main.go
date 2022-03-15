package requestexternal

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requestexternal -destination ./mock_requestexternal.go -source main.go -build_flags=-mod=mod

import (
	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/requestexternal/models/telnyx"
)

// telnyx
const (
	telnyxToken              string = "KEY017B6ED1E90D8FC5DB6ED95F1ACFE4F5_WzTaTxsXJCdwOviG4t1xMM"
	telnyxConnectionID       string = "1762151958791063062"
	telnyxMessagingProfileID string = "40017f8e-49bd-4f16-9e3d-ef103f916228"
)

// twilio
//nolint:deadcode,unused,varcheck // reserved
const (
	twilioSID   string = "AC3300cb9426b78c9ce48db86a755166f0"
	twilioToken string = "58c603e14220f52553be7769b209f423"
)

var (
	metricsNamespace = "number_manager"

	promRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "request_external_process_time",
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

// RequestExternal intreface for ARI request handler
type RequestExternal interface {

	// telnyx
	TelnyxAvailableNumberGets(countryCode, locality, administrativeArea string, limit uint) ([]*telnyx.AvailableNumber, error)

	TelnyxNumberOrdersPost(numbers []string) (*telnyx.OrderNumber, error)

	TelnyxPhoneNumbersGet(size uint, tag, number string) ([]*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDGet(id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDDelete(id string) (*telnyx.PhoneNumber, error)
}

type requestExternal struct{}

// NewRequestExternal create RequestExternal
func NewRequestExternal() RequestExternal {
	h := &requestExternal{}

	return h
}
