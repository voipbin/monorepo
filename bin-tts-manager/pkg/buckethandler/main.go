package buckethandler

//go:generate mockgen -package buckethandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
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
func NewBucketHandler(credentialBase64 string, projectID string, bucketName string, osMediaBucketDirectory string, osAddress string) BucketHandler {
	log := logrus.WithFields(logrus.Fields{
		"func":                      "NewBucketHandler",
		"credential_path":           credentialBase64,
		"project_id":                projectID,
		"bucket_name":               bucketName,
		"os_media_bucket_directory": osMediaBucketDirectory,
		"os_address":                osAddress,
	})
	log.Debugf("Creating a new bucket handler.")

	ctx := context.Background()

	decodedCredential, err := base64.StdEncoding.DecodeString(credentialBase64)
	if err != nil {
		log.Errorf("Error decoding base64 credential: %v", err)
		return nil
	}

	creds, err := google.CredentialsFromJSON(ctx, decodedCredential, storage.ScopeFullControl)
	if err != nil {
		log.Errorf("Could not create credentials from json. err: %v", err)
		return nil
	}

	// parse service account
	conf, err := google.JWTConfigFromJSON(decodedCredential)
	if err != nil {
		logrus.Errorf("Could not parse the credential file. err: %v", err)
		return nil
	}

	// create client
	client, err := storage.NewClient(ctx, option.WithTokenSource(creds.TokenSource))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	tmpAddress := strings.ReplaceAll(osAddress, ".", "-")
	osLocalAddress := fmt.Sprintf("%s.bin-manager.pod.cluster.local", tmpAddress)
	log.Debugf("Generated os local address. os_local_address: %s", osLocalAddress)

	h := &bucketHandler{
		client: client,

		projectID:  projectID,
		bucketName: bucketName,
		accessID:   conf.Email,
		privateKey: conf.PrivateKey,

		osBucketDirectory: osMediaBucketDirectory,
		osLocalAddress:    osLocalAddress,
	}

	return h
}
