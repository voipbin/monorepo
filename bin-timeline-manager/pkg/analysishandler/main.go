package analysishandler

//go:generate mockgen -package analysishandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-timeline-manager/models/analysis"
	"monorepo/bin-timeline-manager/pkg/analysisdbhandler"
	"monorepo/bin-timeline-manager/pkg/eventhandler"
)

// AnalysisHandler orchestrates on-demand AI analysis of ended activeflows and
// owns the persisted analysis record (in its own MySQL store).
//
// Start is the synchronous trigger: it validates ownership + ended-state,
// applies the existing-record policy, creates/resets the progressing row, and
// kicks an async multi-stage LLM chain. Reads/Delete enforce ownership.
type AnalysisHandler interface {
	// Start triggers (or returns an in-flight/existing) analysis for an ended activeflow.
	Start(ctx context.Context, customerID, activeflowID uuid.UUID, reanalyze bool) (*analysis.Analysis, error)
	// Get returns an analysis by id, ownership-checked (masked not-found).
	Get(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)
	// GetByActiveflowID returns the live analysis for an activeflow, ownership-checked.
	GetByActiveflowID(ctx context.Context, customerID, activeflowID uuid.UUID) (*analysis.Analysis, error)
	// List returns a paginated list, always filtered by customer_id (authority).
	List(ctx context.Context, customerID uuid.UUID, pageToken string, pageSize uint64, filters map[analysis.Field]any) ([]*analysis.Analysis, error)
	// Delete soft-deletes (archive-then-delete), ownership-checked.
	Delete(ctx context.Context, customerID, id uuid.UUID) (*analysis.Analysis, error)
}

// StageModels names the LLM models for each analysis stage (and the small
// combined call, which reuses stage3 — review H3). All must be in the gateway
// allow-set.
type StageModels struct {
	Stage1 string // cheap (inventory)
	Stage2 string // capable (content)
	Stage3 string // best (diagnosis / combined)
}

const (
	// analysisMaxEvents caps the canonical event list so a pathological flow
	// cannot exhaust memory (review H1).
	analysisMaxEvents = 5000
	// analysisMaxPages caps the AggregatedList page loop (review H1).
	analysisMaxPages = 100
	// analysisEventPageSize is the per-page fetch size for AggregatedList.
	analysisEventPageSize = 500

	// analysisStageThresholdEvents decides single-call vs full 3-stage chain.
	analysisStageThresholdEvents = 50
	// analysisShortTranscriptRunes: above this combined transcript length the
	// full chain is used even for few events.
	analysisShortTranscriptRunes = 4000

	// analysisReduceTargetBytes is the timeline-side reduce target. It MUST be
	// strictly smaller than the gateway's input cap minus prompt+schema overhead
	// (review M3). The gateway enforces its own hard cap independently.
	analysisReduceTargetBytes = 96 * 1024
	// analysisMaxTranscriptRunesPerResource caps per-resource transcript text in reduction.
	analysisMaxTranscriptRunesPerResource = 8000

	// analysisReanalyzeCooldown gates repeated manual reanalyze on one activeflow (Q7).
	analysisReanalyzeCooldown = 1 * time.Minute

	// analysisJobTimeout bounds the whole async chain.
	analysisJobTimeout = 5 * time.Minute
	// analysisFinalWriteTimeout bounds the final persist on a fresh context so the
	// result is written even if the job ctx is gone.
	analysisFinalWriteTimeout = 10 * time.Second
	// analysisGatewayTimeoutMS is the per-stage gateway RPC timeout (ms).
	analysisGatewayTimeoutMS = 120000

	// analysisMaxConcurrentJobs bounds in-flight async chains (semaphore).
	analysisMaxConcurrentJobs = 8

	// analysisMaxProgressingPerCustomer caps the number of in-flight (progressing)
	// analyses a single customer can have at once (design F1 — cost/DoS guard on
	// the now-public trigger). A new-activeflow trigger past this cap returns
	// ErrConcurrencyLimit (HTTP 429). Re-analyze of an existing row and idempotent
	// returns do NOT count against this (they do not increase concurrency).
	analysisMaxProgressingPerCustomer = 20
)

type analysisHandler struct {
	utilHandler  utilhandler.UtilHandler
	reqHandler   requesthandler.RequestHandler
	dbHandler    analysisdbhandler.AnalysisDBHandler
	eventHandler eventhandler.EventHandler

	models StageModels

	// sem bounds concurrent async chains.
	sem chan struct{}

	// metrics
	metricStarted   prometheus.Counter
	metricCompleted *prometheus.CounterVec
	metricDuration  prometheus.Histogram
}

var (
	promAnalysisStarted = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "timeline_manager",
		Subsystem: "analysis",
		Name:      "started_total",
		Help:      "Total number of analysis chains started.",
	})
	promAnalysisCompleted = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "timeline_manager",
		Subsystem: "analysis",
		Name:      "completed_total",
		Help:      "Total number of analysis chains finished, by terminal status.",
	}, []string{"status"})
	promAnalysisDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "timeline_manager",
		Subsystem: "analysis",
		Name:      "duration_seconds",
		Help:      "Duration of the async analysis chain in seconds.",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300},
	})
)

// NewAnalysisHandler creates a new AnalysisHandler.
func NewAnalysisHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler analysisdbhandler.AnalysisDBHandler,
	eventHandler eventhandler.EventHandler,
	models StageModels,
) AnalysisHandler {
	return &analysisHandler{
		utilHandler:  utilhandler.NewUtilHandler(),
		reqHandler:   reqHandler,
		dbHandler:    dbHandler,
		eventHandler: eventHandler,
		models:       models,
		sem:          make(chan struct{}, analysisMaxConcurrentJobs),

		metricStarted:   promAnalysisStarted,
		metricCompleted: promAnalysisCompleted,
		metricDuration:  promAnalysisDuration,
	}
}
