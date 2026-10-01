package builderhandler

//go:generate mockgen -package builderhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/builder"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

// BuilderHandler is the conversational Assistant Builder (VOIP-1558).
//
// It is stateless: the client sends the whole conversation each turn and
// nothing is stored. Chat makes one model call per request.
type BuilderHandler interface {
	// Chat runs one turn. The error is always a *VoipbinError the caller can
	// return as is; it never carries the customer's input.
	Chat(ctx context.Context, customerID uuid.UUID, req *builder.ChatRequest) (*builder.ChatResponse, error)

	// Status reports whether the Builder is usable and its input limits.
	Status() *builder.StatusResponse
}

// Options are the operational settings of the handler. Config (turn.go) is the
// per-turn model behaviour; Options is what surrounds a turn: the key, the daily limit and the concurrency cap.
type Options struct {
	KeyConfigured bool // false makes Chat fail with BUILDER_UNAVAILABLE and Status report available=false
	DailyLimit    int  // per-customer turns per 24 hour window
	MaxConcurrent int  // per-process cap of calls running at once
}

type builderHandler struct {
	sender Sender
	cache  cachehandler.CacheHandler
	cfg    Config
	opts   Options

	// sem is the per-process concurrency cap. Acquisition never blocks: a full
	// semaphore is an immediate BUILDER_BUSY, so a slow provider cannot pile up
	// requests and starve the shared RPC workers (design 4.3).
	sem chan struct{}
}

var (
	metricsNamespace = "ai_manager"

	// promBuilderChatTotal counts Chat results. The label values are fixed
	// strings; neither the customer id nor any conversation text is ever a label.
	promBuilderChatTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "builder_chat_total",
			Help:      "Total number of Assistant Builder chat turns by result.",
		},
		[]string{"result"},
	)

	// promBuilderChatDuration observes a turn's latency, from entering Chat to returning.
	promBuilderChatDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "builder_chat_duration_seconds",
			Help:      "Duration of Assistant Builder chat turns.",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 20, 30, 40, 60},
		},
	)

	// promBuilderTokensTotal counts model tokens the platform paid for.
	promBuilderTokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "builder_tokens_total",
			Help:      "Total model tokens used by the Assistant Builder, by kind (prompt, completion).",
		},
		[]string{"kind"},
	)
)

func init() {
	prometheus.MustRegister(
		promBuilderChatTotal,
		promBuilderChatDuration,
		promBuilderTokensTotal,
	)
}

// Result label values of ai_manager_builder_chat_total.
const (
	resultOK              = "ok"
	resultDailyLimit      = "daily_limit"
	resultBusy            = "busy"
	resultUnavailable     = "unavailable"
	resultInvalidResponse = "invalid_response"
	resultLLMError        = "llm_error"
	// resultInvalidArgument is not in design 4.8's list. A request that fails
	// validation is the customer's mistake and costs nothing; counting it apart
	// keeps it out of the llm_error and unavailable rates operators watch.
	resultInvalidArgument = "invalid_argument"
	// resultInternal is a panic that listenhandler recovered. The call may have
	// been counted against the daily limit already, so it must show up here.
	resultInternal = "internal"
)

// RecordPanic counts a Builder request whose handling panicked. It is for the
// listen handler's recover, which cannot see the handler's own metrics. The
// label is a fixed word; the panic value, which can carry the input, is not
// an argument on purpose.
func RecordPanic() {
	promBuilderChatTotal.WithLabelValues(resultInternal).Inc()
}

// NewBuilderHandler creates the handler.
//
// MaxConcurrent below one is raised to one: config validation rejects it at
// startup, but a semaphore of size zero would refuse every call for ever, so
// the constructor never builds one.
func NewBuilderHandler(sender Sender, cache cachehandler.CacheHandler, cfg Config, opts Options) BuilderHandler {
	if opts.MaxConcurrent < 1 {
		opts.MaxConcurrent = 1
	}
	return &builderHandler{
		sender: sender,
		cache:  cache,
		cfg:    cfg,
		opts:   opts,
		sem:    make(chan struct{}, opts.MaxConcurrent),
	}
}
