package requestexternal

//go:generate mockgen -package requestexternal -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/pkg/requestexternal/models/telnyx"
	"monorepo/bin-number-manager/pkg/requestexternal/models/twilio"
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
	TelnyxAvailableNumberGets(token, countryCode, locality, administrativeArea string, limit uint) ([]*telnyx.AvailableNumber, error)
	TelnyxNumberOrdersPost(token string, numbers []string, connectionID, profileID string) (*telnyx.OrderNumber, error)
	TelnyxPhoneNumbersGet(token string, size uint, tag, number string) ([]*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersGetByNumber(token string, number string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDGet(token, id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDDelete(token, id string) (*telnyx.PhoneNumber, error)
	TelnyxPhoneNumbersIDUpdate(token, id string, data map[string]interface{}) (*telnyx.PhoneNumber, error)

	// twilio
	TwilioAvailableNumberGets(sid, token, countryCode string, limit uint) ([]*twilio.AvailableNumber, error)
	TwilioIncomingPhoneNumbersPost(sid, token, num, trunkSID string) (*twilio.IncomingPhoneNumber, error)
	TwilioIncomingPhoneNumbersIDDelete(sid, token, id string) error
}

type requestExternal struct{}

// NewRequestExternal create RequestExternal
func NewRequestExternal() RequestExternal {
	h := &requestExternal{}

	return h
}
