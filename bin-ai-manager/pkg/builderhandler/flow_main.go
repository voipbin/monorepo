package builderhandler

//go:generate mockgen -package builderhandler -destination ./mock_flow_main.go -source flow_main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/flowbuilder"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

// FlowBuilderHandler is the conversational Flow AI Builder (VOIP-1573). Like
// the Assistant Builder it is stateless: the client sends the whole
// conversation and the current draft each turn and nothing is stored.
type FlowBuilderHandler interface {
	// Chat runs one turn. The error is always a *VoipbinError the caller can
	// return as is; it never carries the customer's input.
	Chat(ctx context.Context, customerID uuid.UUID, req *flowbuilder.ChatRequest) (*flowbuilder.ChatResponse, error)
}

type flowBuilderHandler struct {
	sender Sender
	cache  cachehandler.CacheHandler
	cfg    Config
	opts   Options

	// sem is the per-process concurrency cap, SHARED with the Assistant
	// Builder (design doc 5.1): the cap protects the platform's LLM provider
	// and is one resource. Acquisition never blocks.
	sem chan struct{}
}

// Flow Builder metrics. Names and labels of the Assistant Builder's metrics
// are untouched; these are separate series with a flow_builder_ prefix so a
// dashboard can tell the two apart (design doc 5.1).
var (
	promFlowBuilderChatTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "flow_builder_chat_total",
			Help:      "Total number of Flow Builder chat turns by result.",
		},
		[]string{"result"},
	)
	promFlowBuilderChatDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "flow_builder_chat_duration_seconds",
			Help:      "Duration of Flow Builder chat turns.",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 20, 30, 40, 60},
		},
	)
	promFlowBuilderTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "flow_builder_tokens_total",
			Help:      "Total model tokens used by the Flow Builder, by kind (prompt, completion).",
		},
		[]string{"kind"},
	)
)

func init() {
	prometheus.MustRegister(
		promFlowBuilderChatTotal,
		promFlowBuilderChatDuration,
		promFlowBuilderTokensTotal,
	)
}

// RecordFlowPanic counts a Flow Builder request whose handling panicked. It
// is for the listen handler's recover, which cannot see this package's
// metrics. The label is a fixed word; the panic value is not an argument on
// purpose.
func RecordFlowPanic() {
	promFlowBuilderChatTotal.WithLabelValues(resultInternal).Inc()
}

// NewBuilderHandlers creates the Assistant and Flow handlers with ONE shared
// concurrency semaphore. MaxConcurrent below one is raised to one.
func NewBuilderHandlers(sender Sender, cache cachehandler.CacheHandler, cfg Config, opts Options) (BuilderHandler, FlowBuilderHandler) {
	if opts.MaxConcurrent < 1 {
		opts.MaxConcurrent = 1
	}
	sem := make(chan struct{}, opts.MaxConcurrent)

	assistant := &builderHandler{sender: sender, cache: cache, cfg: cfg, opts: opts, sem: sem}
	flow := &flowBuilderHandler{sender: sender, cache: cache, cfg: FlowConfig(cfg), opts: opts, sem: sem}
	return assistant, flow
}
