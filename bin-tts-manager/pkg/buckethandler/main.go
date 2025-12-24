package buckethandler

//go:generate mockgen -package buckethandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// BucketHandler intreface for GCP bucket handler
type BucketHandler interface {
	OSFileExist(ctx context.Context, target string) bool
	OSGetFilepath(ctx context.Context, target string) string
	OSGetMediaFilepath(ctx context.Context, target string) string
}

type bucketHandler struct {
	osBucketDirectory string
	osLocalAddress    string // os local address
}

var (
	metricsNamespace = "tts_manager"

	promBucketUploadProcessTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bucket_upload_process_time",
			Help:      "Process time of bucket file upload.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 10000,
			},
		},
	)

	promBucketURLProcessTime = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "bucket_url_process_time",
			Help:      "Process time of download url generate.",
			Buckets: []float64{
				50, 100, 500, 1000, 3000, 10000,
			},
		},
	)
)

func init() {
	prometheus.MustRegister(
		promBucketUploadProcessTime,
		promBucketURLProcessTime,
	)
}

// NewBucketHandler create bucket handler
func NewBucketHandler(osMediaBucketDirectory string, osAddress string) BucketHandler {
	log := logrus.WithFields(logrus.Fields{
		"func":                      "NewBucketHandler",
		"os_media_bucket_directory": osMediaBucketDirectory,
		"os_address":                osAddress,
	})
	log.Debugf("Creating a new bucket handler.")

	tmpAddress := strings.ReplaceAll(osAddress, ".", "-")
	osLocalAddress := fmt.Sprintf("%s.bin-manager.pod.cluster.local", tmpAddress)
	log.Debugf("Generated os local address. os_local_address: %s", osLocalAddress)

	h := &bucketHandler{
		osBucketDirectory: osMediaBucketDirectory,
		osLocalAddress:    osLocalAddress,
	}

	return h
}
