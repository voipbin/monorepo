package streaminghandler

//go:generate mockgen -package streaminghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sync"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-tts-manager/models/streaming"
)

// StreamingHandler define
type StreamingHandler interface {
	Run() error

	Start(
		ctx context.Context,
		customerID uuid.UUID,
		activeflowID uuid.UUID,
		referenceType streaming.ReferenceType,
		referenceID uuid.UUID,
		language string,
		gender streaming.Gender,
		direction streaming.Direction,
	) (*streaming.Streaming, error)
	StartWithID(
		ctx context.Context,
		id uuid.UUID,
		customerID uuid.UUID,
		referenceType streaming.ReferenceType,
		referenceID uuid.UUID,
		language string,
		provider string,
		voiceID string,
		direction streaming.Direction,
	) (*streaming.Streaming, error)
	Stop(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error)

	SayInit(ctx context.Context, id uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error)
	SayAdd(ctx context.Context, id uuid.UUID, messageID uuid.UUID, text string) error
	SayFlush(ctx context.Context, id uuid.UUID) error
	SayStop(ctx context.Context, id uuid.UUID) error
	SayFinish(ctx context.Context, id uuid.UUID, messageID uuid.UUID) (*streaming.Streaming, error)
}

// list of default external media channel options.
//
//nolint:deadcode,varcheck
const (
	defaultEncapsulation  = string(cmexternalmedia.EncapsulationAudioSocket)
	defaultTransport      = string(cmexternalmedia.TransportTCP)
	defaultConnectionType = "client"
	defaultFormat         = "slin" // 8kHz, 16bit, mono signed linear PCM
)

const (
	defaultKeepAliveInterval = 10 * time.Second // 10 seconds
	defaultMaxRetryAttempts  = 3
	defaultInitialBackoff    = 100 * time.Millisecond // 100 milliseconds
)

const (
	variableElevenlabsVoiceID = "voipbin.tts.elevenlabs.voice_id"
)

type streamer interface {
	Init(ctx context.Context, st *streaming.Streaming) (any, error)
	Run(vendorConfig any) error

	SayStop(vendorConfig any) error
	SayAdd(vendorConfig any, text string) error
	SayFlush(vendorConfig any) error
	SayFinish(vendorConfig any) error
}

var (
	metricsNamespace = "tts_manager"

	// streaming_created_total counts new streaming sessions by vendor.
	promStreamingCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_created_total",
			Help:      "Total number of streaming sessions created by vendor.",
		},
		[]string{"vendor"},
	)

	// streaming_ended_total counts ended streaming sessions by vendor.
	promStreamingEndedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_ended_total",
			Help:      "Total number of streaming sessions ended by vendor.",
		},
		[]string{"vendor"},
	)

	// streaming_active tracks currently active streaming sessions by vendor.
	// This resets to 0 on service restart and does not reflect persistent state.
	promStreamingActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_active",
			Help:      "Number of currently active streaming sessions by vendor.",
		},
		[]string{"vendor"},
	)

	// streaming_duration_seconds measures the duration of streaming sessions by vendor.
	promStreamingDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_duration_seconds",
			Help:      "Duration of streaming sessions in seconds by vendor.",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300},
		},
		[]string{"vendor"},
	)

	// streaming_message_total counts Say messages (SayInit calls).
	promStreamingMessageTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_message_total",
			Help:      "Total number of streaming Say messages initiated.",
		},
	)

	// streaming_error_total counts streaming errors by vendor.
	promStreamingErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_error_total",
			Help:      "Total number of streaming errors by vendor.",
		},
		[]string{"vendor"},
	)

	// streaming_language_total counts streaming sessions by language and gender.
	promStreamingLanguageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "streaming_language_total",
			Help:      "Total number of streaming sessions by language and gender.",
		},
		[]string{"language", "gender"},
	)
)

func init() {
	prometheus.MustRegister(
		promStreamingCreatedTotal,
		promStreamingEndedTotal,
		promStreamingActive,
		promStreamingDurationSeconds,
		promStreamingMessageTotal,
		promStreamingErrorTotal,
		promStreamingLanguageTotal,
	)
}

type streamingHandler struct {
	utilHandler    utilhandler.UtilHandler
	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler

	listenAddress string
	podID         string

	mapStreaming map[uuid.UUID]*streaming.Streaming
	muStreaming  sync.Mutex

	elevenlabsHandler streamer
}

// NewStreamingHandler define
func NewStreamingHandler(
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	listenAddress string,
	podID string,
	elevenlabsAPIKey string,
) StreamingHandler {

	elevenlabsHandler := NewElevenlabsHandler(reqHandler, notifyHandler, elevenlabsAPIKey)

	return &streamingHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		requestHandler: reqHandler,
		notifyHandler:  notifyHandler,

		listenAddress: listenAddress,
		podID:         podID,

		mapStreaming: make(map[uuid.UUID]*streaming.Streaming),
		muStreaming:  sync.Mutex{},

		elevenlabsHandler: elevenlabsHandler,
	}
}
