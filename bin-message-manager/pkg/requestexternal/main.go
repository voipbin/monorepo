package requestexternal

//go:generate mockgen -package requestexternal -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-message-manager/models/messagebird"
	"monorepo/bin-message-manager/models/telnyx"
)

var (
	metricsNamespace = "message_manager"

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

	// messagebird
	MessagebirdSendMessage(ctx context.Context, sender string, destinations []string, text string) (*messagebird.Message, error)

	// telnyx
	TelnyxSendMessage(ctx context.Context, source string, destination string, text string) (*telnyx.MessageResponse, error)
}

type requestExternal struct {
	authtokenMessagebird string // authentication token for messagebird
	authtokenTelnyx      string // authentication token for telnyx
}

// NewRequestExternal create RequestExternal
func NewRequestExternal(authtokenMessagebird string, authtokenTelnyx string) RequestExternal {
	h := &requestExternal{
		authtokenMessagebird: authtokenMessagebird,
		authtokenTelnyx:      authtokenTelnyx,
	}

	return h
}
