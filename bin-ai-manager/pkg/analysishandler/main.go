package analysishandler

//go:generate mockgen -package analysishandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-ai-manager/models/analysis"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// AnalysisHandler is the generic, stateless, internal-only LLM gateway.
//
// It exposes a single operation: take {prompt, data, schema, schema_name, model?}
// and run one structured-output (json_schema, strict) chat completion, returning
// the schema-conformant JSON plus accounting. It is NOT exposed via api-manager /
// OpenAPI / RST; only internal managers (e.g. timeline-manager) reach it over RPC.
type AnalysisHandler interface {
	Run(ctx context.Context, req *analysis.Request) (*analysis.Response, error)
}

type analysisHandler struct {
	utilHandler utilhandler.UtilHandler

	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler

	defaultModel  string
	allowedModels map[string]bool
	maxInputBytes int
	maxOutputToks int
	// reasoningEffort, when non-empty, is sent as reasoning_effort on the chat
	// request. "none" disables Gemini 2.5 "thinking" so the token budget is spent
	// on the JSON output, not internal reasoning (prevents finish_reason=length
	// truncation on large staged analyses).
	reasoningEffort string
}

var (
	metricsNamespace = "ai_manager"

	// analysis_gateway_run_total counts gateway runs by model.
	promAnalysisGatewayRunTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "analysis_gateway_run_total",
			Help:      "Total number of analysis gateway runs by model.",
		},
		[]string{"model"},
	)

	// analysis_gateway_run_duration_seconds observes gateway latency by model.
	promAnalysisGatewayRunDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "analysis_gateway_run_duration_seconds",
			Help:      "Duration of analysis gateway runs by model.",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 20, 30, 60, 120},
		},
		[]string{"model"},
	)
)

func init() {
	prometheus.MustRegister(
		promAnalysisGatewayRunTotal,
		promAnalysisGatewayRunDuration,
	)
}

// NewAnalysisHandler creates the gateway handler.
//
// allowedModels MUST be a superset of every model any internal caller (e.g.
// timeline-manager's stage models) will request, otherwise a requested model is
// silently coerced to defaultModel.
func NewAnalysisHandler(
	engineOpenaiHandler engine_openai_handler.EngineOpenaiHandler,
	defaultModel string,
	allowedModels []string,
	maxInputBytes int,
	maxOutputTokens int,
	reasoningEffort string,
) AnalysisHandler {
	allowed := map[string]bool{}
	for _, m := range allowedModels {
		allowed[m] = true
	}
	// the default is always allowed.
	allowed[defaultModel] = true

	return &analysisHandler{
		utilHandler: utilhandler.NewUtilHandler(),

		engineOpenaiHandler: engineOpenaiHandler,

		defaultModel:    defaultModel,
		allowedModels:   allowed,
		maxInputBytes:   maxInputBytes,
		maxOutputToks:   maxOutputTokens,
		reasoningEffort: reasoningEffort,
	}
}
