package buckethandler

//go:generate mockgen -package buckethandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
)

// BucketHandler intreface for GCP bucket handler
type BucketHandler interface {
	FileGet(ctx context.Context, target string) ([]byte, error)
	FileGetDownloadURL(target string, expire time.Time) (string, error)
	FileExist(ctx context.Context, target string) bool
	FileUpload(ctx context.Context, src string, dest string) error

	GetBucketName() string

	OSFileExist(ctx context.Context, target string) bool
	OSGetFilepath(ctx context.Context, target string) string
	OSGetMediaFilepath(ctx context.Context, target string) string
}

type bucketHandler struct {
	client *storage.Client

	projectID  string
	bucketName string
	accessID   string
	privateKey []byte

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
func NewBucketHandler(projectID string, bucketName string, osMediaBucketDirectory string, osAddress string) BucketHandler {
	log := logrus.WithFields(logrus.Fields{
		"func":                      "NewBucketHandler",
		"project_id":                projectID,
		"bucket_name":               bucketName,
		"os_media_bucket_directory": osMediaBucketDirectory,
		"os_address":                osAddress,
	})
	log.Debugf("Creating a new bucket handler.")

	var client *storage.Client
	var accessID string
	var privateKey []byte
	var errClient error
	ctx := context.Background()

	envCredPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if envCredPath != "" {
		log.Infof("Found GOOGLE_APPLICATION_CREDENTIALS at: %s", envCredPath)

		jsonContent, err := os.ReadFile(envCredPath)
		if err != nil {
			log.Errorf("Failed to read credential file: %v", err)
			return nil
		}

		conf, err := google.JWTConfigFromJSON(jsonContent)
		if err != nil {
			log.Errorf("Failed to parse credential JSON: %v", err)
			return nil
		}

		accessID = conf.Email
		privateKey = conf.PrivateKey
		client, errClient = storage.NewClient(ctx)
	} else {
		log.Info("No GOOGLE_APPLICATION_CREDENTIALS, trying ADC/Metadata")

		client, errClient = storage.NewClient(ctx)
		privateKey = nil
		if metadata.OnGCE() {
			log.Debugf("The service is running on the GCE")
			email, err := metadata.EmailWithContext(ctx, "default")
			if err != nil {
				log.Errorf("Failed to retrieve service account email from metadata: %v", err)
			} else {
				accessID = email
			}
		} else {
			log.Warn("Could not determine Service Account Email (Not on GCE/GKE)")
		}
	}

	if errClient != nil {
		log.Errorf("Failed to create client: %v", errClient)
		return nil
	}
	log.Debugf("Checking account. project_id: %s, access_id: %s", projectID, accessID)

	tmpAddress := strings.ReplaceAll(osAddress, ".", "-")
	osLocalAddress := fmt.Sprintf("%s.bin-manager.pod.cluster.local", tmpAddress)
	log.Debugf("Generated os local address. os_local_address: %s", osLocalAddress)

	h := &bucketHandler{
		client: client,

		projectID:  projectID,
		bucketName: bucketName,
		accessID:   accessID,
		privateKey: privateKey,

		osBucketDirectory: osMediaBucketDirectory,
		osLocalAddress:    osLocalAddress,
	}

	return h
}
