package builderhandler

//go:generate mockgen -package builderhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

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

	// Status reports whether the Builder is usable and its input limits. The key is read at the
	// time of the call; a key that can not be read is reported as available=false.
	Status(ctx context.Context) *builder.StatusResponse
}

// Options are the operational settings of the handler. Config (turn.go) is the
// per-turn model behaviour; Options is what surrounds a turn: the key, the daily limit and the concurrency cap.
type Options struct {
	// KeyConfigured tells, at the time of the call, whether the platform key the Builder needs has
	// a value. It is read on every Chat and Status (the key can be changed while the service
	// runs). false makes Chat fail with BUILDER_UNAVAILABLE and Status report available=false; an
	// error means the key could not be read, which is not the same as "not configured": Chat
	// stops before it counts the turn. A nil function means "not configured".
	KeyConfigured func(ctx context.Context) (bool, error)
	DailyLimit    int // per-customer turns per 24 hour window
	MaxConcurrent int // per-process cap of calls running at once
}

// keyCheckTimeout bounds the key check that Chat and Status run before anything else. Chat is
// called with a context that has no deadline, and the key read of the model call has its own
// budget inside the call deadline (at most 50 seconds), so this check is the only extra time. It
// must stay short: 2 seconds plus the longest call deadline is below the 55 seconds that
// api-manager waits (see config.Validate for the 5 seconds of headroom this uses).
const keyCheckTimeout = 2 * time.Second

// keyConfigured runs the key check of the options within keyCheckTimeout.
func (o Options) keyConfigured(ctx context.Context) (bool, error) {
	if o.KeyConfigured == nil {
		return false, nil
	}

	cctx, cancel := context.WithTimeout(ctx, keyCheckTimeout)
	defer cancel()

	return o.KeyConfigured(cctx)
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
	resultOK          = "ok"
	resultDailyLimit  = "daily_limit"
	resultBusy        = "busy"
	resultUnavailable = "unavailable"
	// resultKeyError is a platform key that could not be read. The environment read of the platform
	// keys does not fail, so the suppliers do not return this today; the label stays so that the
	// metric series do not change. It is kept apart from
	// unavailable (the key is not configured) and llm_error (the provider answered).
	resultKeyError        = "key_error"
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
