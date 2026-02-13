package ttshandler

//go:generate mockgen -package ttshandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/audiohandler"
	"monorepo/bin-tts-manager/pkg/buckethandler"
)

// TTSHandler intreface for tts handler
type TTSHandler interface {
	Create(ctx context.Context, callID uuid.UUID, text string, lang string, provider tts.Provider, voiceID string) (*tts.TTS, error)
}

type ttsHandler struct {
	audioHandler  audiohandler.AudioHandler
	bucketHandler buckethandler.BucketHandler

	requestHandler requesthandler.RequestHandler
	notifyHandler  notifyhandler.NotifyHandler
}

var (
	metricsNamespace = "tts_manager"

	promHashProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "hash_process_time",
			Help:      "Process time of hash gererate.",
			Buckets: []float64{
				50, 100, 500, 1000,
			},
		},
		[]string{},
	)

	// speech_request_total counts batch TTS requests by result (cache_hit, created, error).
	promSpeechRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "speech_request_total",
			Help:      "Total number of batch TTS speech requests by result.",
		},
		[]string{"result"},
	)

	// speech_create_duration_seconds measures end-to-end audio creation latency (cache misses only).
	promSpeechCreateDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "speech_create_duration_seconds",
			Help:      "Duration of batch TTS audio creation in seconds (cache misses only).",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{},
	)

	// speech_language_total counts batch TTS requests by language and provider.
	promSpeechLanguageTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "speech_language_total",
			Help:      "Total number of batch TTS requests by language and provider.",
		},
		[]string{"language", "provider"},
	)

	// speech_fallback_total counts how often fallback to an alternative provider occurs.
	promSpeechFallbackTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "speech_fallback_total",
			Help:      "Total number of batch TTS fallbacks by original provider.",
		},
		[]string{"from_provider"},
	)
)

func init() {
	prometheus.MustRegister(
		promHashProcessTime,
		promSpeechRequestTotal,
		promSpeechCreateDurationSeconds,
		promSpeechLanguageTotal,
		promSpeechFallbackTotal,
	)
}

// NewTTSHandler create TTSHandler
func NewTTSHandler(
	awsAccessKey string,
	awsSecretKey string,
	mediaBucketDirectory string,
	localAddress string,
	requestHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) TTSHandler {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "NewTTSHandler",
		"media_bucket_directory": mediaBucketDirectory,
		"local_address":          localAddress,
	})
	log.Debugf("Creating a new TTSHandler.")

	ctx := context.Background()
	audioHandler := audiohandler.NewAudioHandler(ctx, awsAccessKey, awsSecretKey)
	if audioHandler == nil {
		log.Errorf("Could not create audio handler.")
		return nil
	}

	bucketHandler := buckethandler.NewBucketHandler(mediaBucketDirectory, localAddress)
	if bucketHandler == nil {
		log.Errorf("Could not create bucket handler.")
		return nil
	}

	h := &ttsHandler{
		audioHandler:  audioHandler,
		bucketHandler: bucketHandler,

		requestHandler: requestHandler,
		notifyHandler:  notifyHandler,
	}

	return h
}
