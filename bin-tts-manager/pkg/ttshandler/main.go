package ttshandler

//go:generate mockgen -package ttshandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/audiohandler"
	"monorepo/bin-tts-manager/pkg/buckethandler"
)

// TTSHandler intreface for tts handler
type TTSHandler interface {
	Create(ctx context.Context, callID uuid.UUID, text string, lang string, gender tts.Gender) (*tts.TTS, error)
}

type ttsHandler struct {
	credentailPath string

	audioHandler  audiohandler.AudioHandler
	bucketHandler buckethandler.BucketHandler

	notifyHandler notifyhandler.NotifyHandler
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
)

func init() {
	prometheus.MustRegister(
		promHashProcessTime,
	)
}

// NewTTSHandler create TTSHandler
func NewTTSHandler(
	credentialBase64 string,
	projectID string,
	bucketName string,
	mediaBucketDirectory string,
	localAddress string,
	notifyHandler notifyhandler.NotifyHandler,
) TTSHandler {
	log := logrus.WithFields(logrus.Fields{
		"func":                   "NewTTSHandler",
		"project_id":             projectID,
		"bucket_name":            bucketName,
		"media_bucket_directory": mediaBucketDirectory,
		"local_address":          localAddress,
	})
	log.Debugf("Creating a new TTSHandler.")

	ctx := context.Background()
	audioHandler := audiohandler.NewAudioHandler(ctx, credentialBase64)
	if audioHandler == nil {
		log.Errorf("Could not create audio handler.")
		return nil
	}

	bucketHandler := buckethandler.NewBucketHandler(credentialBase64, projectID, bucketName, mediaBucketDirectory, localAddress)
	if bucketHandler == nil {
		log.Errorf("Could not create bucket handler.")
		return nil
	}

	h := &ttsHandler{
		credentailPath: credentialBase64,

		audioHandler:  audioHandler,
		bucketHandler: bucketHandler,

		notifyHandler: notifyHandler,
	}

	return h
}
