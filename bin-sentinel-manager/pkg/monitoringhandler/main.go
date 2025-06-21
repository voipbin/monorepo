package monitoringhandler

import (
	"context"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
)

// list of namespaces
const (
	namespaceVOIP = "voip"
	namespaceBIN  = "bin"
)

// list of lables
const (
	lableAppAsteriskCall = "asterisk-call"
)

type monitoringHandler struct {
	util          utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

type MonitoringHandler interface {
	Run(ctx context.Context, selectors map[string][]string) error
}

func NewMonitoringHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
	utilHandler utilhandler.UtilHandler,
) MonitoringHandler {
	h := &monitoringHandler{
		util:          utilHandler,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}

	return h
}

var (
	metricsNamespace = commonoutline.GetMetricNameSpace(commonoutline.ServiceNameSentinelManager)

	promPodStateChangeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "pod_state_change_total",
			Help:      "Counts the number of pod state changes",
		},
		[]string{"namespace", "pod", "state"},
	)
)

func init() {
	prometheus.MustRegister(
		promPodStateChangeCounter,
	)
}
