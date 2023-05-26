package buckethandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package buckethandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"io/ioutil"
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
}

type bucketHandler struct {
	client *storage.Client

	projectID  string
	bucketName string
	accessID   string
	privateKey []byte
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
func NewBucketHandler(credentialPath string, projectID string, bucketName string) BucketHandler {

	ctx := context.Background()

	jsonKey, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		logrus.Errorf("Could not read the credential file. err: %v", err)
		return nil
	}

	// parse service account
	conf, err := google.JWTConfigFromJSON(jsonKey)
	if err != nil {
		logrus.Errorf("Could not parse the credential file. err: %v", err)
		return nil
	}

	// create client
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialPath))
	if err != nil {
		logrus.Errorf("Could not create a new client. err: %v", err)
		return nil
	}

	h := &bucketHandler{
		client: client,

		projectID:  projectID,
		bucketName: bucketName,
		accessID:   conf.Email,
		privateKey: conf.PrivateKey,
	}

	return h
}
