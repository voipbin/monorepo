package requestexternal

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package requestexternal -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-message-manager/models/messagebird"
)

// // messagebird
// const (
// 	messagebirdAuth string = "AccessKey tEIpkNHtIzO0FR4RBsWfEOrce"
// )

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
	MessagebirdSendMessage(sender string, destinations []string, text string) (*messagebird.Message, error)
}

type requestExternal struct {
	authtokenMessagebird string // authentication token for messagebird
}

// NewRequestExternal create RequestExternal
func NewRequestExternal(authtokenMessagebird string) RequestExternal {
	h := &requestExternal{
		authtokenMessagebird: authtokenMessagebird,
	}

	return h
}
